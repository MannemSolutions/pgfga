package pg

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/jackc/pgx/v4"
	"strings"
)

type Roles map[string]Role

type Role struct {
	handler *Handler
	name string
	options []string
}

func NewRole(handler *Handler, name string, options []string) (r *Role, err error) {
	role, exists := handler.roles[name]
	if exists {
		log.Debugf("We must make sure that options are added when passed in here")
		return &role, nil
	}
	r = &Role{
		handler: handler,
		name: name,
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
	if ! ph.strictOptions.Users {
		log.Infof("not dropping user/role %s (config.strict.roles is not True)", r.name)
		return nil
	}
	existsQuery := "SELECT rolname FROM pg_roles WHERE rolname = $1 AND rolname != CURRENT_USER"
	exists, err := ph.runQueryExists(existsQuery, r.name)
	if err != nil {
		return err
	}
	if ! exists {
		delete(r.handler.roles, r.name)
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
		dbHandler := ph.GetDb(dbname).GetDbHandler()
		err = dbHandler.runQueryExec("REASSIGN OWNED BY {} TO {}", identifier(r.name), identifier(newOwner))
		if err != nil {
			return err
		}
	}
	err = ph.runQueryExec("DROP ROLE {}", identifier(r.name))
	if err != nil {
		return err
	}
	delete(r.handler.roles, r.name)
	log.Infof("Dropped role '%s'", r.name)
	return nil
}

func (r Role) Create() (err error) {
	ph := r.handler
	exists, err := ph.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = $1", r.name)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("create role %s", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Created role '%s'", r.name)
	}
	var invalidOptions []string
	for _, option := range r.options {
		err = r.setRoleOption(option)
		if err == InvalidOption {
			invalidOptions = append(invalidOptions, option)
		}
	}
	if len(invalidOptions) > 0 {
		return fmt.Errorf("creating role %s with invalid role options (%s)", r.name,
			strings.Join(invalidOptions, ", "))
	}
	return nil
}

func (r Role) setRoleOption(option string) (err error) {
	ph := r.handler
	option = strings.ToUpper(option)
	if optionSql, ok := validRoleOptions[option]; ! ok {
		exists, err := ph.runQueryExists("SELECT rolname FROM pg_roles WHERE rolname = $1 AND " + optionSql, r.name)
		if err != nil {
			return err
		}
		if ! exists {
			log.Debugf("setRoleOption ALTER %s with %s", r.name, option)
			err = ph.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH " + option, identifier(r.name)))
			if err != nil {
				return err
			}
		}
	} else  {
		return InvalidOption
	}
	return nil
}

func (r Role) GrantRole(grantedRole *Role) (err error) {
	ph := r.handler
	checkQry := `select granted.rolname granted_role 
		from pg_auth_members auth inner join pg_roles 
		granted on auth.roleid = granted.oid inner join pg_roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2`
	exists, err := ph.runQueryExists(checkQry, grantedRole.name, r.name)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(grantedRole.name), identifier(r.name)))
		if err != nil {
			return err
		}
	}
	log.Infof("Granted role '%s' to user '%s'", grantedRole.name, r.name)
	return nil
}

func (r Role) RevokeRole(roleName string) (err error) {
	ph := r.handler
	checkQry := `select granted.rolname granted_role, grantee.rolname 
		grantee_role from pg_auth_members auth inner join pg_roles 
		granted on auth.roleid = granted.oid inner join pg_roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2 and grantee.rolname != CURRENT_USER`
	exists, err := ph.runQueryExists(checkQry, roleName, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = ph.runQueryExec(fmt.Sprintf("REVOKE %s FROM %s", identifier(roleName), identifier(r.name)))
		if err != nil {
			return err
		}
	}
	log.Infof("Revoked role '%s' from user '%s'", roleName, r.name)
	return nil
}

func (r Role) SetPassword(userName string, password string) (err error) {
	ph := r.handler
	var hashedPassword string
	if len(password) == 35 && strings.HasPrefix(password, "md5") {
		hashedPassword = password
	} else {
		hashedPassword = fmt.Sprintf("md5%x", md5.Sum([]byte(password+userName)))
	}
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1 AND COALESCE(passwd, '') != $2`
	exists, err := ph.runQueryExists(checkQry, userName, hashedPassword)
	if err != nil {
		return err
	}
	if ! exists {
		err = ph.runQueryExec(fmt.Sprintf("ALTER USER %s WITH ENCRYPTED PASSWORD $1", identifier(userName)),
			hashedPassword)
		if err != nil {
			return err
		}
	}
	return nil
}
func (r Role) ResetPassword(userName string) (err error) {
	ph := r.handler
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1
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