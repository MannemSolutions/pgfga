package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type Databases map[string]Database

type Database struct {
	defaultHandler *Handler
	dbHandler *Handler
	name string
	owner string
	extensions Extensions
}

func NewDatabase(handler *Handler, name string, owner string) (d *Database) {
	db, exists := handler.databases[name]
	if exists {
		log.Debugf("We must make sure that the owner is checked here too")
		return &db
	}
	if owner == "" {
		owner = name
	}
	d = &Database{
		defaultHandler: handler,
		name: name,
		owner: owner,
		extensions: make(Extensions),
	}
	handler.databases[name] = *d
	return d
}

func (d *Database) GetDbHandler() (h *Handler) {
	if d.dbHandler != nil {
		return d.dbHandler
	}
	// not yet initialized. Let's initialize
	if d.defaultHandler.DbName() == d.name {
		d.dbHandler = d.defaultHandler
		return d.dbHandler
	}

	var connParams map[string]string
	for key, value := range d.defaultHandler.connParams {
		connParams[key] = value
	}
	connParams["dbname"] = d.name
	d.dbHandler = NewPgHandler(connParams, d.defaultHandler.strictOptions)
	return d.dbHandler
}

func (d Database) Drop() (err error) {
	ph := d.defaultHandler
	if ph.strictOptions.Users {
		log.Infof("skipping drop of database %s (not running with strict option for databases", d.name)
		return nil
	}
	exists, err := ph.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if exists {
		return ph.runQueryExec(fmt.Sprintf("drop database %s", identifier(d.name)))
	}
	return nil
}

func (d Database) Create() (err error) {
	ph := d.defaultHandler

	exists, err := ph.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Created database '%s'", d.name)
	}
	exists, err = ph.runQueryExists("SELECT datname FROM pg_database db inner join pg_roles rol on db.datdba = rol.oid WHERE datname = $1 and rolname = $2", d.name, d.owner)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(d.name), identifier(d.owner)))
		if err != nil {
			return err
		}
		log.Infof("Altered database owner on '%s' to '%s'", d.name, d.owner)
	}
	err = ph.GrantRole(d.owner, "opex")
	if err != nil {
		return err
	}
	readOnlyRoleName := fmt.Sprintf("%s_readonly", d.name)
	err = ph.GrantRole(readOnlyRoleName, "readonly")
	if err != nil {
		return err
	}
	return d.SetReadOnlyGrants(readOnlyRoleName)
}

func (d Database) SetReadOnlyGrants(readOnlyRoleName string) (err error) {
	ph := d.GetDbHandler()
	err = ph.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tableswhere schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (SELECT table_schema||'.'||table_name 
              FROM information_schema.role_table_grants WHERE grantee = $1 and privilege_type = 'SELECT')`
	row := ph.conn.QueryRow(context.Background(), query, readOnlyRoleName)
	for {
		scanErr := row.Scan(&schema)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		schemas = append(schemas, schema)
	}
	for _, schema := range schemas {
		err = ph.runQueryExec(fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s", identifier(schema),
			identifier(readOnlyRoleName)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) AddExtension(name string, schema string, version string) (e *Extension) {
	e = NewExtension(d, name , schema , version )
	d.extensions[name] = *e
	return e
}

func (d *Database) CreateExtensions() (err error) {
	for _, e := range d.extensions {
		err = e.Create()
		if err != nil {
			return err
		}
	}
	return nil
}