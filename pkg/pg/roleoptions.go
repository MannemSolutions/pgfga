package pg

import (
	"fmt"
	"strings"
)

type RoleOption struct {
	name    string
	sql     string
	enabled bool
}

func NewRoleOption(name string) (opt RoleOption, err error) {
	opt.name = strings.ToUpper(name)
	if strings.HasPrefix(opt.name, "NO") {
		opt.name = opt.name[2:]
		opt.enabled = false
	} else {
		opt.enabled = true
	}
	if sql, exists := ValidRoleOptions[name]; exists {
		opt.sql = sql
		return opt, nil
	}
	var validRoleOptionNames []string
	for _, oName := range ValidRoleOptions {
		validRoleOptionNames = append(validRoleOptionNames, oName)
	}
	return opt, fmt.Errorf("invalid RoleOption %s (should fit to re `NO(%s)`)", name, strings.Join(validRoleOptionNames, "|"))
}

func (opt RoleOption) Valid() (isValid bool) {
	return opt.String() != ""
}

func (opt RoleOption) String() (name string) {
	name = strings.ToUpper(opt.name)
	if _, exists := ValidRoleOptions[name]; !exists {
		return ""
	}
	if opt.enabled {
		return name
	}
	return fmt.Sprintf("NO%s", name)
}

func (opt RoleOption) Sql() (sql string) {
	if opt.enabled {
		return opt.sql
	}
	return fmt.Sprintf("not %s", opt.sql)
}
func (opt RoleOption) Inverse() (invOpt RoleOption) {
	return RoleOption{
		name:    opt.name,
		sql:     opt.sql,
		enabled: !opt.enabled,
	}
}

var (
	ValidRoleOptions = map[string]string{
		"SUPERUSER":   "rolsuper",
		"CREATEROLE ": "rolcreaterole",
		"CREATEUSER":  "rolcreaterole",
		"INHERIT ":    "rolinherit",
		"LOGIN":       "rolcanlogin",
		"REPLICATION": "rolreplication",
	}
)

// MarshalYAML marshals the enum as a quoted json string
func (opt RoleOption) MarshalYAML() (interface{}, error) {
	return opt.String(), nil
}

// UnmarshalYAML converts a yaml string to the enum value
func (opt *RoleOption) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	tmpOpt, err := NewRoleOption(name)
	if err != nil {
		return err
	}
	opt.name = tmpOpt.name
	opt.sql = tmpOpt.sql
	opt.enabled = tmpOpt.enabled
	return nil
}

type RoleOptions map[string]RoleOption

//func (ros RoleOptions)Join(sep string) (joined string) {
//	var strOptions []string
//	for _, option := range ros {
//		strOptions = append(strOptions, string(option))
//	}
//	return strings.Join(strOptions, sep)
//}

var (
	loginOption, _ = NewRoleOption("LOGIN")
	LoginOptions   = RoleOptions{loginOption.name: loginOption}
	//EmptyOptions RoleOptions
)
