package pg

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"
)

var log *zap.SugaredLogger

func Initialize(logger *zap.SugaredLogger) {
	log = logger
}

var InvalidOption = errors.New("invalid role option")

type Dsn map[string]string

type StrictOptions struct {
	Users      bool `yaml:"users"`
	Databases  bool `yaml:"databases"`
	Extensions bool `yaml:"extensions"`
	Slots      bool `yaml:"replication_slots"`
}

// identifier returns the object name ready to be used in a sql query as an object name (e.a. select * from %s)
func identifier(objectName string) (escaped string) {
	return fmt.Sprintf("\"%s\"", strings.Replace(objectName, "\"", "\"\"", -1))
}

// quotedSqlValue uses proper quoting for values in SQL queries
func quotedSqlValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.Replace(objectName, "'", "''", -1))
}

// connectStringValue uses proper quoting for connect string values
func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.Replace(objectName, "'", "\\'", -1))
}
