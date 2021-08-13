package pg

import "strings"

type RoleOption string

func (opt RoleOption) Valid() bool {
	return opt.Name() != ""
}

func (opt RoleOption) Name() (name string) {
	name = strings.ToUpper(string(opt))
	if _, exists := ValidRoleOptions[RoleOption(name)]; exists {
		return name
	}
	return ""
}

func (opt RoleOption) SqlOption() (sql string) {
	name := strings.ToUpper(string(opt))
	if val, exists := ValidRoleOptions[RoleOption(name)]; exists {
		return val
	}
	return ""
}

var (
	ValidRoleOptions = map[RoleOption]string{"SUPERUSER": "rolsuper",
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
)

type RoleOptions []RoleOption

func (ros RoleOptions)Join(sep string) (joined string) {
	var strOptions []string
	for _, option := range ros {
		strOptions = append(strOptions, string(option))
	}
	return strings.Join(strOptions, sep)
}