package internal

import (
	"fmt"
	"github.com/mannemsolutions/pgfga/pkg/ldap"
	"github.com/mannemsolutions/pgfga/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var (
	log *zap.SugaredLogger
	atom zap.AtomicLevel
)
func Initialize() {
	atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	)).Sugar()

	pg.Initialize(log)
	ldap.Initialize(log)
}

type PgFgaHandler struct {
	config FgaConfig
	pg     *pg.Handler
	ldap   *ldap.Handler
}

func NewPgFgaHandler() (pfh *PgFgaHandler, err error) {
	config, err := NewConfig()
	if err != nil {
		return pfh, err
	}

	atom.SetLevel(config.GeneralConfig.LogLevel)

	pfh = &PgFgaHandler{
		config: config,
	}

	ldapUser, err := config.LdapConfig.User()
	if err != nil {
		return pfh, err
	}

	ldapPassword, err := config.LdapConfig.Password()
	if err != nil {
		return pfh, err
	}

	pfh.ldap = ldap.NewLdapHandler(config.LdapConfig.Servers, ldapUser, ldapPassword, config.LdapConfig.MaxRetries)

	pfh.pg = pg.NewPgHandler(config.PgConfig.KeyPairs(), config.StrictConfig)

	return pfh, nil
}

func (pfh PgFgaHandler) Handle() {
	err := pfh.HandleUsers()
	if err != nil {
		log.Fatal(err)
	}
}

func (pfh PgFgaHandler) HandleUsers() (err error) {
	for userName, userConfig := range pfh.config.UserConfig {
		switch userConfig.Auth {
		case "ldap-group":
			log.Debugf("Configuring role from ldap for %s", userName)
			if userConfig.BaseDN == "" || userConfig.Filter == "" {
				return fmt.Errorf("ldapbasedn and ldapfilter must be set for %s (auth: 'ldap-group')", userName)
			}
			baseGroup, err := pfh.ldap.GetMembers(userConfig.BaseDN, userConfig.Filter)
			if err != nil {
				return err
			}
			for _, ms := range baseGroup.MembershipTree() {
				err = pfh.pg.GrantRole(ms.Member.Name(), ms.MemberOf.Name())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
