package internal

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
	Users     bool `yaml:"users"`
	Databases bool `yaml:"databases"`
}

type FgaLdapConfig struct {
	BaseDN       string   `yaml:"basedn"`
	user         string   `yaml:"user"`
	userFile     string   `yaml:"userfile"`
	password     string   `yaml:"password"`
	passwordFile string   `yaml:"passwordfile"`
	base64       bool     `yaml:"base64"`
	Servers      []string `yaml:"servers"`
	MaxRetries   int      `yaml:"conn_retries"`
}

func isExecutable(filename string) (isExecutable bool, err error) {
	fi, err := os.Lstat("some-filename")
	if err != nil {
		return false, err
	}
	mode := fi.Mode()
	return mode&0111 == 0111, nil
}

func fromExecutable(filename string) (value string, err error) {
	out, err := exec.Command(filename).Output()
	if err != nil {
		return "", nil
	}
	return string(out), nil
}

func fromFile(filename string) (value string, err error) {
	isExec, err := isExecutable(filename)
	if isExec {
		return fromExecutable(filename)
	}
	file, err := os.Open(filename) // For read access.
	if err != nil {
		return "", err
	}
	data := make([]byte, 100)
	count, err := file.Read(data)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("file %s is empty", filename)
	}
	return string(data[:]), nil
}

func (flc FgaLdapConfig) User() (user string, err error) {
	if flc.user != "" {
		user = flc.user
	} else if flc.userFile != "" {
		user, err = fromFile(flc.userFile)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("missing ldap user name (either user or userfile must be set)")
	}
	if flc.base64 {
		data, err := base64.StdEncoding.DecodeString(user)
		if err != nil {
			return "", err
		}
		user = string(data)
	}
	return user, nil
}

func (flc FgaLdapConfig) Password() (password string, err error) {
	if flc.password != "" {
		password = flc.password
	} else if flc.passwordFile != "" {
		password, err = fromFile(flc.passwordFile)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("missing ldap password (either password or passwordfile must be set)")
	}
	if flc.base64 {
		data, err := base64.StdEncoding.DecodeString(password)
		if err != nil {
			return "", err
		}
		password = string(data)
	}
	return password, nil
}

type FgaPostgresConfig struct {
    dsn map[string]string `yaml:"dsn"`
}

func (fpc FgaPostgresConfig) DSN() (dsn string) {
	var pairs []string
	for key, value := range fpc.dsn {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(pairs[:], " ")
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
