package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger
func Initialize(logger *zap.SugaredLogger) {
	log = logger
}

type Handler struct {
	connString string
	conn *pgx.Conn
}

func NewPgHandler(connString string) (ph *Handler) {
	return &Handler{
		connString: connString,
	}
}

func (ph *Handler) Connect() (err error) {
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

func (ph *Handler) runQueryGetOneField(query string) (answer string, err error) {
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

func (ph *Handler) GrantRole(user string, role string) (err error) {
	log.Infof("GRANT %s to %s", role, user)
	return nil
}
