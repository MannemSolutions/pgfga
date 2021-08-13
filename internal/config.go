package internal

import (
	"github.com/mannemsolutions/pgfga/pkg/ldap"
	"github.com/mannemsolutions/pgfga/pkg/pg"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"os"
	"path/filepath"
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
	LogLevel zapcore.Level `yaml:"loglevel"`
	RunDelay int           `yaml:"run_delay"`
}

type FgaPostgresConfig struct {
	Dsn pg.Dsn `yaml:"dsn"`
}

type FgaUserConfig struct {
	Auth     string    `yaml:"auth"`
	BaseDN   string    `yaml:"ldapbasedn"`
	Filter   string    `yaml:"ldapfilter"`
	MemberOf []string  `yaml:"memberof"`
	Expiry   time.Time `yaml:"expiry"`
	Password string    `yaml:"password"`
}

type FgaRoles struct {
	Options  []pg.RoleOption `yaml:"options"`
	MemberOf []string        `yaml:"member"`
}

type FgaConfig struct {
	GeneralConfig FgaGeneralConfig         `yaml:"general"`
	StrictConfig  pg.StrictOptions         `yaml:"strict"`
	LdapConfig    ldap.Config              `yaml:"ldap"`
	PgConfig      FgaPostgresConfig        `yaml:"postgresql"`
	DbsConfig     pg.Databases             `yaml:"databases"`
	UserConfig    map[string]FgaUserConfig `yaml:"users"`
	Debug         bool                     `yaml:"debug"`
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

	// This only parsed as yaml, nothing else
	// #nosec
	yamlConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	return config, err
}
