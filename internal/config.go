package internal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

/*
 * This module reads the config file and returns a config object with all entries from the config yaml file.
 */

const (
	envConfName     = "PGFGACONFIG"
	defaultConfFile = "./pgfgaconfig.yaml"
)

type FgaGeneralConfig struct {
	LogLevel  string `yaml:"loglevel"`
	RunDelay int     `yaml:"run_delay"`
}

type FgaStrictConfig struct {
	Users     string `yaml:"users"`
	Databases int    `yaml:"databases"`
}

type FgaLdapConfig struct {
	BaseDN       string   `yaml:"basedn"`
	PasswordFile string   `yaml:"passwordfile"`
	UserFile     string   `yaml:"userfile"`
	Servers      []string `yaml:"servers"`
}

type FgaPostgresConfig struct {
    Dsn map[string]string `yaml:"dsn"`
}

type FgaExtensionConfig struct {
	Schema string `yaml:"schema"`
	State string `yaml:"state"`
	Version float32 `yaml:"version"`
}

type FgaDbConfig struct {
	State string `yaml:"state"`
	Extensions map[string]FgaExtensionConfig `yaml:"extensions"`
}

type FgaUserConfig struct {
	Auth string `yaml:"auth"`
	BaseDN string `yaml:"ldapbasedn"`
	Filter string `yaml:"ldapfilter"`
	MemberOf []string `yaml:"memberof"`
	Expiry time.Time `yaml:"expiry"`
	Password string `yaml:"password"`
}

var roleOptions = map[string]string{"SUPERUSER": "rolsuper",
"NOSUPERUSER": "not rolsuper",
"NOCREATEDB": "not rolcreatedb",
"CREATEROLE ": "rolcreaterole",
"NOCREATEROLE": "not rolcreaterole",
"CREATEUSER ": "rolcreaterole",
"NOCREATEUSER": "not rolcreaterole",
"INHERIT ": "rolinherit",
"NOINHERIT": "not rolinherit",
"LOGIN": "rolcanlogin",
"NOLOGIN": "not rolcanlogin",
"REPLICATION": "rolreplication",
"NOREPLICATION": "not rolreplication",
}

type FgaRoleOptions string

func (opt FgaRoleOptions) Valid() bool {
	return opt.Name() != ""
}

func (opt FgaRoleOptions) Name() (name string) {
	name = strings.ToUpper(string(opt))
	if _, ok := roleOptions[name]; ok {
		return name
	}
	return ""
}

func (opt FgaRoleOptions) SqlOption() (sql string) {
	sql, _ = roleOptions[strings.ToUpper(string(opt))]
	return
}

type FgaRoles struct {
	Options  []FgaRoleOptions `yaml:"options"`
	MemberOf []string `yaml:"member"`
}

type FgaConfig struct {
	GeneralConfig        FgaGeneralConfig               `yaml:"general"`
	StrictConfig       FgaStrictConfig                     `yaml:"strict"`
	LdapConfig    FgaLdapConfig `yaml:"ldap"`
	PgConfig FgaPostgresConfig                       `yaml:"postgresql"`
	DbsConfig map[string]FgaDbConfig `yaml:"databases"`
	UserConfig map[string]FgaUserConfig `yaml:"users"`
	Debug      bool                       `yaml:"debug"`
}

func NewConfig() (config FgaConfig, err error) {
	configFile := os.Getenv(envConfName)
	if configFile == "" {
		configFile = defaultConfFile
	}
	configFile, err = filepath.EvalSymlinks(configFile)
	if err != nil {
		return config, err
	}

	yamlConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	return config, err
}
