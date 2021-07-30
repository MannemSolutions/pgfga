package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type PgHandler struct {
	connString string
	conn *pgx.Conn
}

func NewPgHandler(connString string) (ph *PgHandler) {
	return &PgHandler{
		connString: connString,
	}
}

func (ph *PgHandler) Connect() (err error) {
	if ph.conn != nil {
		if ph.conn.IsClosed() {
			ph.conn = nil
		} else {
			return nil
		}
	}
	ph.conn, err = pgx.Connect(context.Background(), ph.connString)
	if err != nil {
		ph.conn = nil
		return err
	}
	return nil
}

func (ph *PgHandler) runQueryGetOneField(query string) (answer string, err error) {
	err = ph.Connect()
	if err != nil {
		return "", err
	}

	err = ph.conn.QueryRow(context.Background(), query).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("runQueryGetOneField (%s) failed: %v\n", query, err)
	}
	return answer, nil
}

func (ph *PgHandler) GrantRole(user string, role string) (err error) {
	fmt.Sprintf("GRANT %s to %s", role, user)
	return nil
}
