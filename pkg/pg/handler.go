package pg

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/jackc/pgx/v4"
	"os"
	"os/user"
	"strings"
)

type Handler struct {
	connParams map[string]string
	conn *pgx.Conn
	strictOptions StrictOptions
	dbHandlers map[string]Handler
}

func NewPgHandler(connParams map[string]string, options StrictOptions) (ph *Handler) {
	return &Handler{
		connParams: connParams,
		strictOptions: options,
	}
}

func (ph *Handler) DbName() (dbName string) {
	value, ok := ph.connParams["dbname"]
	if ok {
		return value
	}
	value = os.Getenv("PGDATABASE")
	if value != "" {
		return value
	}
	return ph.UserName()
}

func (ph *Handler) UserName() (userName string) {
	value, ok := ph.connParams["user"]
	if ok {
		return value
	}
	value = os.Getenv("PGUSER")
	if value != "" {
		return value
	}
	currentUser, err := user.Current()
	if err != nil {
		panic("cannot determine current user")
	}
	return currentUser.Username
}

func (ph *Handler) DSN() (dsn string) {
	var pairs []string
	for key, value := range ph.connParams {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, connectStringValue(value)))
	}
	return strings.Join(pairs[:], " ")
}

func (ph *Handler) Connect() (err error) {
	if ph.conn != nil {
		if ph.conn.IsClosed() {
			ph.conn = nil
		} else {
			return nil
		}
	}
	ph.conn, err = pgx.Connect(context.Background(), ph.DSN())
	if err != nil {
		ph.conn = nil
		return err
	}
	return nil
}

func (ph *Handler) getDbHandler(dbName string) (h *Handler) {
	if ph.DbName() == dbName {
		return ph
	}

	handler, ok := ph.dbHandlers[dbName]
	if ok {
		return &handler
	}

	var connParams map[string]string
	for key, value := range ph.connParams {
		connParams[key] = value
	}
	connParams["dbname"] = dbName
	h = NewPgHandler(connParams, ph.strictOptions)
	ph.dbHandlers[dbName] = *h
	return h
}

func (ph *Handler) runQueryExists(query string, args ...interface{}) (answer bool, err error) {
	err = ph.Connect()
	if err != nil {
		return false, err
	}
	err = ph.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return true, nil
}

func (ph *Handler) runQueryExec(query string, args ...interface{}) (err error) {
	err = ph.Connect()
	if err != nil {
		return err
	}
	_, err = ph.conn.Exec(context.Background(), query, args...)
	return err
}

func (ph *Handler) runQueryGetOneField(query string, args ...interface{}) (answer string, err error) {
	err = ph.Connect()
	if err != nil {
		return "", err
	}

	err = ph.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("runQueryGetOneField (%s) failed: %v\n", query, err)
	}
	return answer, nil
}

func (ph *Handler) DropDB(dbName string) (err error) {
	if ph.strictOptions.Users {
		log.Infof("skipping drop of database %s (not running with strict option for databases")
		return nil
	}
	exists, err := ph.runQueryExists("SELECT datname FROM pg_database WHERE datname = %s", dbName)
	if err != nil {
		return err
	}
	if exists {
		return ph.runQueryExec(fmt.Sprintf("drop database %s", identifier(dbName)))
	}
	return nil
}

func (ph *Handler) CreateDB(dbName string, ownerName string) (err error) {
	if ownerName == "" {
		ownerName = dbName
	}

	exists, err := ph.runQueryExists("SELECT datname FROM pg_database WHERE datname = %s", dbName)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(dbName)))
		if err != nil {
			return err
		}
		log.Infof("Created database '%s'", dbName)
	}
	exists, err = ph.runQueryExists("SELECT datname FROM pg_database db inner join pg_roles rol on db.datdba = rol.oid WHERE datname = %s and rolname = %s", dbName, ownerName)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(dbName), identifier(ownerName)))
		if err != nil {
			return err
		}
		log.Infof("Altered database owner on '%s' to '%s'", dbName, ownerName)
	}
	err = ph.GrantRole(ownerName, "opex")
	if err != nil {
		return err
	}
	readOnlyRoleName := fmt.Sprintf("%s_readonly")
	err = ph.GrantRole(readOnlyRoleName, "readonly")
	if err != nil {
		return err
	}
	dbHandler := ph.getDbHandler(dbName)
	return dbHandler.setReadOnlyGrants(readOnlyRoleName)
}

func (ph Handler) setReadOnlyGrants(readOnlyRoleName string) (err error) {
	err = ph.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tableswhere schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (SELECT table_schema||'.'||table_name 
              FROM information_schema.role_table_grants WHERE grantee = %s and privilege_type = 'SELECT')`
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

func (ph *Handler) DropRole(roleName string) (err error) {
	if ! ph.strictOptions.Users {
		log.Infof("not dropping user/role %s (config.strict.roles is not True)", roleName)
		return nil
	}
	existsQuery := "SELECT rolname FROM pg_roles WHERE rolname = %s AND rolname != CURRENT_USER"
	exists, err := ph.runQueryExists(existsQuery, roleName)
	if err != nil {
		return err
	}
	if ! exists {
		return nil
	}
	var dbname string
	var newOwner string
	query := `select db.datname, o.rolname as newOwner from pg_database db inner join 
			  pg_roles o on db.datdba = o.oid where db.datname != 'template0'`
	row := ph.conn.QueryRow(context.Background(), query)
	for {
		scanErr := row.Scan(&dbname, &newOwner)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		dbHandler := ph.getDbHandler(dbname)
		err = dbHandler.runQueryExec("REASSIGN OWNED BY {} TO {}", identifier(roleName), identifier(newOwner))
		if err != nil {
			return err
		}
	}
	err = ph.runQueryExec("DROP ROLE {}", identifier(roleName))
	if err != nil {
		return err
	}
	log.Infof("Dropped role '%s'", roleName)
	return nil
}

func (ph *Handler) CreateRole(roleName string, options []string) (err error) {
	exists, err := ph.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = %s", roleName)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("create role %s", identifier(roleName)))
		if err != nil {
			return err
		}
		log.Infof("Created role '%s'", roleName)
	}
	var invalidOptions []string
	for _, option := range options {
		err = ph.setRoleOption(roleName, option)
		if err == InvalidOption {
			invalidOptions = append(invalidOptions, option)
		}
	}
	if len(invalidOptions) > 0 {
		return fmt.Errorf("creating role %s with invalid role options (%s)", roleName,
			strings.Join(invalidOptions, ", "))
	}
	return nil
}

func (ph *Handler) setRoleOption(roleName string, option string) (err error) {
	option = strings.ToUpper(option)
	if optionSql, ok := validRoleOptions[option]; ! ok {
		exists, err := ph.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = %s AND " + optionSql, roleName)
		if err != nil {
			return err
		}
		if ! exists {
			log.Debugf("setRoleOption ALTER %s with %s", roleName, option)
			err = ph.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH " + option, identifier(roleName)))
			if err != nil {
				return err
			}
		}
	} else  {
		return InvalidOption
	}
	return nil
}

func (ph *Handler) GrantRole(userName string, roleName string) (err error) {
	err = ph.CreateRole(userName, []string{})
	if err != nil {
		return err
	}
	err = ph.CreateRole(roleName, []string{})
	if err != nil {
		return err
	}
	checkQry := `select granted.rolname granted_role, grantee.rolname 
		grantee_role from pg_auth_members auth inner join pg_roles 
		granted on auth.roleid = granted.oid inner join pg_roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = %s and grantee.rolname = %s`
	exists, err := ph.runQueryExists(checkQry, roleName, userName)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(roleName), identifier(userName)))
		if err != nil {
			return err
		}
	}
	log.Infof("Granted role '%s' to user '%s'", roleName, userName)
	return nil
}

func (ph *Handler) SetPassword(userName string, password string) (err error) {
	var hashedPassword string
	if len(password) == 35 && strings.HasPrefix(password, "md5") {
		hashedPassword = password
	} else {
		hashedPassword = fmt.Sprintf("md5%x", md5.Sum([]byte(password+userName)))
	}
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = %s
	AND COALESCE(passwd, '') != %s`
	exists, err := ph.runQueryExists(checkQry, userName, hashedPassword)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("ALTER USER %s WITH ENCRYPTED PASSWORD %s", identifier(userName),
			quotedSqlValue(hashedPassword)))
		if err != nil {
			return err
		}
	}
	return nil
}
func (ph *Handler) ResetPassword(userName string) (err error) {
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = %s
	AND Passwd IS NOT NULL AND usename != CURRENT_USER`
	exists, err := ph.runQueryExists(checkQry, userName)
	if err != nil {
		return err
	}
	if exists {
		err = ph.runQueryExec(fmt.Sprintf("ALTER USER %s WITH PASSWORD NULL", identifier(userName)))
		if err != nil {
			return err
		}
	}
	return nil
}
