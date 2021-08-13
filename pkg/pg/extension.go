package pg

import (
	"fmt"
)

type Extensions map[string]Extension

type Extension struct {
	// name and db are set by the database
	db      *Database
	name    string
	Schema  string `yaml:"schema"`
	State   State  `yaml:"state"`
	Version string `yaml:"version"`
}

func NewExtension(db *Database, name string, schema string, version string) (e *Extension, err error) {
	ext, exists := db.Extensions[name]
	if exists {
		if ext.Schema != e.Schema || e.Version != ext.Version {
			return nil, fmt.Errorf("db %s already has extension %s defined, with different schema and/or version",
				db.name, e.name)
		}
		return &ext, nil
	}
	e = &Extension{
		db:      db,
		name:    name,
		Schema:  schema,
		Version: version,
		State:   Present,
	}
	db.Extensions[name] = *e
	return e, nil
}

func (e Extension) Drop() (err error) {
	ph := e.db.handler
	c := e.db.GetDbConnection()
	if !e.db.handler.strictOptions.Extensions {
		log.Infof("not dropping extension '%s'.'%s' (config.strict.roles is not True)", e.db.name, e.name)
		return nil
	}
	dbExistsQuery := "SELECT datname FROM pg_database WHERE datname = $1"
	exists, err := c.runQueryExists(dbExistsQuery, e.db.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	dbConn := ph.GetDb(e.db.name).GetDbConnection()
	err = dbConn.runQueryExec("DROP EXTENSION IF EXISTS " + identifier(e.name))
	if err != nil {
		return err
	}
	delete(e.db.Extensions, e.name)
	log.Infof("Dropped '%s'.'%s'", e.db.name, e.name)
	return nil
}

func (e Extension) Create() (err error) {
	c := e.db.GetDbConnection()
	// First let's see if the extension and version is available
	exists, err := c.runQueryExists("SELECT * FROM pg_available_extensions WHERE name = $1",
		e.name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("extension %s is not available", e.name)
	}
	exists, err = c.runQueryExists("SELECT * FROM pg_available_extension_versions WHERE name = $1 AND version = $2",
		e.name, e.Version)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("version %s is not available for extension %s", e.Version, e.name)
	}
	exists, err = c.runQueryExists("SELECT * FROM pg_extension WHERE name = $1", e.name, e.Version)
	if err != nil {
		return err
	}
	if !exists {
		createQry := "CREATE EXTENSION IF NOT EXISTS " + identifier(e.name)
		if e.Schema != "" {
			createQry += " SCHEMA " + identifier(e.Schema)
		}
		if e.Version != "" {
			createQry += "VERSION" + identifier(e.Version)
		}
		err = c.runQueryExec(createQry)
		if err != nil {
			return err
		}
		log.Infof("Created extension '%s'.'%s'", e.db.name, e.name)
		return nil
	}
	if e.Version == "" {
		return nil
	}
	currentVersion, err := c.runQueryGetOneField("SELECT extversion FROM pg_extension WHERE extname = $1", e.name)
	if err != nil {
		return err
	}
	if currentVersion != e.Version {
		err = c.runQueryExec("ALTER EXTENSION "+identifier(e.name)+" UPDATE TO $1", e.Version)
		if err != nil {
			return err
		}
		log.Infof("Updated extension '%s'.'%s' to version '%s'", e.db.name, e.name, e.Version)
	}
	return nil
}
