package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"os"
	"os/user"
	"strings"
)

type Handler struct {
	connParams Dsn
	conn *pgx.Conn
	strictOptions StrictOptions
	databases Databases
	roles Roles
}

func NewPgHandler(connParams Dsn, options StrictOptions) (ph *Handler) {
	return &Handler{
		connParams: connParams,
		strictOptions: options,
		databases: make(Databases),
		roles: make(Roles),
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

func (ph *Handler) GetDb(dbName string) (d *Database) {
	// NewDatabase does everything we need to do
	return NewDatabase(ph, dbName, "")
}

func (ph *Handler) GetRole(roleName string) (d *Role, err error) {
	// NewDatabase does everything we need to do
	return NewRole(ph, roleName, []string{})
}

func (ph *Handler) GrantRole(granteeName string, grantedName string) (err error) {
	// NewDatabase does everything we need to do
	grantee, err := ph.GetRole(granteeName)
	if err != nil {
		return err
	}
	granted, err := ph.GetRole(grantedName)
	if err != nil {
		return err
	}
	return grantee.GrantRole(granted)
}

func (ph *Handler) runQueryExists(query string, args ...interface{}) (exists bool, err error) {
	err = ph.Connect()
	if err != nil {
		return false, err
	}
	var answer string
	err = ph.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err == nil {
		return true, nil
	}
	return false, err
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

func (ph *Handler) StrictifyRoles() (err error) {
 return nil
}

func (ph *Handler) StrictifyDatabases() (err error) {
	return nil
}
func (ph *Handler) StrictifyExtensions() (err error) {
	return nil
}
