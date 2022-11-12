package internal

import (
	"flag"
	"fmt"
	"github.com/mannemsolutions/pgfga/pkg/ldap"
	"github.com/mannemsolutions/pgfga/pkg/pg"
	"go.uber.org/zap/zapcore"
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
	defaultConfFile = "/etc/pgfga/config.yaml"
)

type FgaGeneralConfig struct {
	LogLevel zapcore.Level `yaml:"loglevel"`
	RunDelay time.Duration `yaml:"run_delay"`
	Debug    bool          `yaml:"debug"`
}

type FgaUserConfig struct {
	Auth     string    `yaml:"auth"`
	BaseDN   string    `yaml:"ldapbasedn"`
	Filter   string    `yaml:"ldapfilter"`
	MemberOf []string  `yaml:"memberof"`
	Options  []string  `yaml:"options"`
	Expiry   time.Time `yaml:"expiry"`
	Password string    `yaml:"password"`
	State    pg.State  `yaml:"state"`
}

type FgaRoleConfig struct {
	Options  []string `yaml:"options"`
	MemberOf []string `yaml:"member"`
	State    pg.State `yaml:"state"`
}

type FgaConfig struct {
	GeneralConfig FgaGeneralConfig         `yaml:"general"`
	StrictConfig  pg.StrictOptions         `yaml:"strict"`
	LdapConfig    ldap.Config              `yaml:"ldap"`
	PgDsn         pg.Dsn                   `yaml:"postgresql_dsn"`
	DbsConfig     pg.Databases             `yaml:"databases"`
	UserConfig    map[string]FgaUserConfig `yaml:"users"`
	Roles         map[string]FgaRoleConfig `yaml:"roles"`
	Slots         []string                 `yaml:"replication_slots"`
}

func NewConfig() (config FgaConfig, err error) {
	var configFile string
	var debug bool
	var version bool
	flag.BoolVar(&debug, "d", false, "Add debugging output")
	flag.BoolVar(&version, "v", false, "Show version information")
	flag.StringVar(&configFile, "c", os.Getenv(envConfName), "Path to configfile")

	flag.Parse()
	if version {
		fmt.Println(appVersion)
		os.Exit(0)
	}
	if configFile == "" {
		configFile = defaultConfFile
	}
	configFile, err = filepath.EvalSymlinks(configFile)
	if err != nil {
		return config, err
	}

	// This only parsed as yaml, nothing else
	// #nosec
	yamlConfig, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	config.GeneralConfig.Debug = config.GeneralConfig.Debug || debug
	return config, err
}
