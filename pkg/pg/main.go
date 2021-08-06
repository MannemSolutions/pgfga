package pg

import (
	"fmt"
	"go.uber.org/zap"
	"strings"
)

var log *zap.SugaredLogger
func Initialize(logger *zap.SugaredLogger) {
	log = logger
}

var InvalidOption error

type StrictOptions struct {
	Users      bool `yaml:"users"`
	Databases  bool `yaml:"databases"`
	Extensions bool `yaml:"extensions"`
}
// identifier returns the object name ready to be used in a sql query as an object name (e.a. select * from %s)
func identifier(objectName string) (escaped string) {
	return fmt.Sprintf("\"%s\"", strings.Replace(objectName, "\"", "\"\"", -1))
}

// connstrDbName returns the db name ready to be used in a connection string
func connstrDbName(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.Replace(objectName, "'", "\\'", -1))
}