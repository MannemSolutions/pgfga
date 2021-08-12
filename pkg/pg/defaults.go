package pg

var (
	ValidRoleOptions = map[string]string{"SUPERUSER": "rolsuper",
		"NOSUPERUSER":   "not rolsuper",
		"NOCREATEDB":    "not rolcreatedb",
		"CREATEROLE ":   "rolcreaterole",
		"NOCREATEROLE":  "not rolcreaterole",
		"CREATEUSER ":   "rolcreaterole",
		"NOCREATEUSER":  "not rolcreaterole",
		"INHERIT ":      "rolinherit",
		"NOINHERIT":     "not rolinherit",
		"LOGIN":         "rolcanlogin",
		"NOLOGIN":       "not rolcanlogin",
		"REPLICATION":   "rolreplication",
		"NOREPLICATION": "not rolreplication",
	}

	ProtectedRoles = map[string]bool{"aq_administrator_role": true,
		"enterprisedb":              true,
		"postgres":                  true,
		"pg_monitor":                true,
		"pg_read_all_settings":      true,
		"pg_read_all_stats":         true,
		"pg_stat_scan_tables":       true,
		"pg_signal_backend":         true,
		"pg_read_server_files":      true,
		"pg_write_server_files":     true,
		"pg_execute_server_program": true,
	}

	ProtectedDatabases = map[string]bool{"postgres": true,
		"template0": true,
		"template1": true,
	}

	LogonOptions = []string{"LOGON"}

	EmptyOptions []string
)
