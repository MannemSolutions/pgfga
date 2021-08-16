package pg

import (
	"context"
	// md5 is weak, but it is still an accepted password algorithm in Postgres.
	// #nosec
	"crypto/md5"
	"fmt"
	"github.com/jackc/pgx/v4"
	"strings"
)

type Roles map[string]Role

type Role struct {
	handler *Handler
	name    string
	options RoleOptions
}

func NewRole(handler *Handler, name string, options RoleOptions) (r *Role, err error) {
	role, exists := handler.roles[name]
	if exists {
		for _, option := range options {
			role.options[option.name] = option
		}
		return &role, nil
	}
	r = &Role{
		handler: handler,
		name:    name,
		options: options,
	}
	err = r.Create()
	if err != nil {
		return r, err
	}
	handler.roles[name] = *r
	return r, nil
}

func (r *Role) Drop() (err error) {
	ph := r.handler
	c := ph.conn
	if !ph.strictOptions.Users {
		log.Infof("not dropping user/role %s (config.strict.roles is not True)", r.name)
		return nil
	}
	existsQuery := "SELECT rolname FROM pg_roles WHERE rolname = $1 AND rolname != CURRENT_USER"
	exists, err := c.runQueryExists(existsQuery, r.name)
	if err != nil {
		return err
	}
	if !exists {
		delete(r.handler.roles, r.name)
		return nil
	}
	var dbname string
	var newOwner string
	query := `select db.datname, o.rolname as newOwner from pg_database db inner join 
			  pg_roles o on db.datdba = o.oid where db.datname != 'template0'`
	row := c.conn.QueryRow(context.Background(), query)
	for {
		scanErr := row.Scan(&dbname, &newOwner)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		dbConn := ph.GetDb(dbname).GetDbConnection()
		err = dbConn.runQueryExec("REASSIGN OWNED BY {} TO {}", identifier(r.name), identifier(newOwner))
		if err != nil {
			return err
		}
		log.Debugf("Reassigned ownership from '%s' to '%s' in db '%s'", r.name, newOwner, dbname)
	}
	err = c.runQueryExec("DROP ROLE {}", identifier(r.name))
	if err != nil {
		return err
	}
	delete(r.handler.roles, r.name)
	log.Infof("Role '%s' succesfully dropped", r.name)
	return nil
}

func (r Role) Create() (err error) {
	c := r.handler.conn
	exists, err := c.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = $1", r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("CREATE ROLE %s", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' succesfully created", r.name)
	}
	for _, option := range r.options {
		err = r.setRoleOption(option)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r Role) setRoleOption(option RoleOption) (err error) {
	c := r.handler.conn
	optionSql := option.Sql()
	exists, err := c.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = $1 AND "+optionSql, r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH "+option.String(), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Debugf("Role '%s' succesfully altered with option '%s'", r.name, option)
	}
	return nil
}

func (r Role) GrantRole(grantedRole *Role) (err error) {
	c := r.handler.conn
	checkQry := `select granted.rolname granted_role 
		from pg_auth_members auth inner join pg_roles 
		granted on auth.roleid = granted.oid inner join pg_roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2`
	exists, err := c.runQueryExists(checkQry, grantedRole.name, r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(grantedRole.name), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' succesfully granted to user '%s'", grantedRole.name, r.name)
	} else {
		log.Debugf("Role '%s' already granted to user '%s'", grantedRole.name, r.name)
	}
	return nil
}

func (r Role) RevokeRole(roleName string) (err error) {
	c := r.handler.conn
	checkQry := `select granted.rolname granted_role, grantee.rolname 
		grantee_role from pg_auth_members auth inner join pg_roles 
		granted on auth.roleid = granted.oid inner join pg_roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2 and grantee.rolname != CURRENT_USER`
	exists, err := c.runQueryExists(checkQry, roleName, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("REVOKE %s FROM %s", identifier(roleName), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' succesfully revoked from user '%s'", roleName, r.name)
	}
	return nil
}

func (r Role) SetPassword(password string) (err error) {
	if password == "" {
		return r.ResetPassword()
	}
	var hashedPassword string
	if len(password) == 35 && strings.HasPrefix(password, "md5") {
		hashedPassword = password
	} else {
		// #nosec
		hashedPassword = fmt.Sprintf("md5%x", md5.Sum([]byte(password+r.name)))
	}
	c := r.handler.conn
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1 AND COALESCE(passwd, '') != $2`
	exists, err := c.runQueryExists(checkQry, r.name, hashedPassword)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER USER %s WITH ENCRYPTED PASSWORD %s", identifier(r.name),
			quotedSqlValue(hashedPassword)))
		if err != nil {
			return err
		}
		log.Infof("Succesfully set new password for user '%s'", r.name)
	}
	return nil
}
func (r Role) ResetPassword() (err error) {
	c := r.handler.conn
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1
	AND Passwd IS NOT NULL AND usename != CURRENT_USER`
	exists, err := c.runQueryExists(checkQry, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER USER %s WITH PASSWORD NULL", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Succesfully removed password for user '%s'", r.name)
	}
	return nil
}
