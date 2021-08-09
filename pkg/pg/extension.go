package pg

import (
	"fmt"
)

type Extensions map[string]Extension

type Extension struct {
	db *Database
	name string
	schema string
	version string
}

func NewExtension(db *Database, name string, schema string, version string) (e *Extension) {
	log.Debugf("Register extension to database, and return when already exists in databse. And check schema and version")
	return &Extension{
		db: db,
		name: name,
		schema: schema,
		version: version,
	}
}

func (e Extension) Drop() (err error) {
	ph := e.db.dbHandler
	if ! ph.strictOptions.Extensions {
		log.Infof("not dropping extension '%s'.'%s' (config.strict.roles is not True)", e.db.name, e.name)
		return nil
	}
	dbExistsQuery := "SELECT datname FROM pg_database WHERE datname = $1"
	exists, err := ph.runQueryExists(dbExistsQuery, e.db.name)
	if err != nil {
		return err
	}
	if ! exists {
		return nil
	}

	dbHandler := ph.GetDb(e.db.name).GetDbHandler()
		err = dbHandler.runQueryExec("DROP EXTENSION IF EXISTS " + identifier(e.name))
		if err != nil {
			return err
		}
	delete(e.db.extensions, e.name)
	log.Infof("Dropped '%s'.'%s'", e.db.name, e.name)
	return nil
}

func (e Extension) Create() (err error) {
	ph := e.db.dbHandler
	// First let's see if the extension and version is available
	exists, err := ph.runQueryExists("SELECT * FROM pg_available_extensions WHERE name = $1",
		e.name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("extension %s is not available", e.name)
	}
	exists, err = ph.runQueryExists("SELECT * FROM pg_available_extension_versions WHERE name = $1 AND version = $2",
		e.name, e.version)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("version %s is not available for extension %s", e.version, e.name)
	}
	exists, err = ph.runQueryExists("SELECT * FROM pg_extension WHERE name = $1", e.name, e.version)
	if err != nil {
		return err
	}
	if !exists {
		createQry := "CREATE EXTENSION IF NOT EXISTS " + identifier(e.name)
		if e.schema != "" {
			createQry += " SCHEMA " + identifier(e.schema)
		}
		if e.version != "" {
			createQry += "VERSION" + identifier(e.version)
		}
		err = ph.runQueryExec(createQry)
		if err != nil {
			return err
		}
		log.Infof("Created extension '%s'.'%s'", e.db.name, e.name)
		return nil
	}
	if e.version == "" {
		return nil
	}
	currentVersion, err := ph.runQueryGetOneField("SELECT extversion FROM pg_extension WHERE extname = $1", e.name)
	if err != nil {
		return err
	}
	if currentVersion != e.version {
		err = ph.runQueryExec("ALTER EXTENSION " + identifier(e.name) + " UPDATE TO $1", e.version)
		if err != nil {
			return err
		}
		log.Infof("Updated extension '%s'.'%s' to version '%s'", e.db.name, e.name, e.version)
	}
	return nil
}