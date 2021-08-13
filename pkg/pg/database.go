package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type Databases map[string]Database

type Database struct {
	// for DB's created from yaml, handler and name are set by the pg.Handler
	handler *Handler
	name    string
	// conn is created from handler when required
	conn       *Conn
	Owner      string     `yaml:"owner"`
	Extensions Extensions `yaml:"extensions"`
	State      string     `yaml:"state"`
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
		handler:    handler,
		name:       name,
		Owner:      owner,
		Extensions: make(Extensions),
	}
	handler.databases[name] = *d
	return d
}

//SetDefaults is called to set all defaults for databases created from yaml
func (d *Database) SetDefaults() {
	for name, ext := range d.Extensions {
		ext.db = d
		ext.name = name
	}
}

func (d *Database) GetDbConnection() (c *Conn) {
	if d.conn != nil {
		return d.conn
	}
	// not yet initialized. Let's initialize
	if d.handler.conn.DbName() == d.name {
		d.conn = d.handler.conn
		return d.conn
	}

	connParams := make(map[string]string)
	for key, value := range d.handler.conn.connParams {
		connParams[key] = value
	}
	connParams["dbname"] = d.name
	d.conn = NewConn(connParams)
	return d.conn
}

func (d Database) Drop() (err error) {
	ph := d.handler
	if ph.strictOptions.Users {
		log.Infof("skipping drop of database %s (not running with strict option for databases", d.name)
		return nil
	}
	exists, err := ph.conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if exists {
		return ph.conn.runQueryExec(fmt.Sprintf("drop database %s", identifier(d.name)))
	}
	return nil
}

func (d Database) Create() (err error) {
	ph := d.handler

	exists, err := ph.conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if !exists {
		err = ph.conn.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Created database '%s'", d.name)
	}
	exists, err = ph.conn.runQueryExists("SELECT datname FROM pg_database db inner join pg_roles rol on db.datdba = rol.oid WHERE datname = $1 and rolname = $2", d.name, d.Owner)
	if err != nil {
		return err
	}
	if !exists {
		err = ph.conn.runQueryExec(fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(d.name), identifier(d.Owner)))
		if err != nil {
			return err
		}
		log.Infof("Altered database owner on '%s' to '%s'", d.name, d.Owner)
	}
	err = ph.GrantRole(d.Owner, "opex")
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
	c := d.GetDbConnection()
	err = c.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tableswhere schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (SELECT table_schema||'.'||table_name 
              FROM information_schema.role_table_grants WHERE grantee = $1 and privilege_type = 'SELECT')`
	row := c.conn.QueryRow(context.Background(), query, readOnlyRoleName)
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
		err = c.runQueryExec(fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s", identifier(schema),
			identifier(readOnlyRoleName)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) AddExtension(name string, schema string, version string) (e *Extension, err error) {
	e, err = NewExtension(d, name, schema, version)
	if err != nil {
		return nil, err
	}
	d.Extensions[name] = *e
	return e, nil
}

func (d *Database) CreateOrDropExtensions() (err error) {
	for _, e := range d.Extensions {
		if e.State == "present" {
			err = e.Create()
		} else {
			err = e.Drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
