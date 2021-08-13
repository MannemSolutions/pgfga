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
	log  *zap.SugaredLogger
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

	pfh.ldap = ldap.NewLdapHandler(config.LdapConfig)

	slots := make(pg.ReplicationSlots)
	for _, slotName := range config.Slots {
		slot := pg.NewSlot(pfh.pg, slotName)
		slots[slotName] = *slot
	}
	pfh.pg = pg.NewPgHandler(config.PgConfig.Dsn, config.StrictConfig, config.DbsConfig, slots)

	return pfh, nil
}

func (pfh PgFgaHandler) Handle() {
	err := pfh.HandleRoles()
	if err != nil {
		log.Fatal(err)
	}
	err = pfh.HandleUsers()
	if err != nil {
		log.Fatal(err)
	}
	err = pfh.HandleDatabases()
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
				_, err := pg.NewRole(pfh.pg, ms.Member.Name(), pg.LogonOptions)
				if err != nil {
					return err
				}
				err = pfh.pg.GrantRole(ms.Member.Name(), baseGroup.Name())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (pfh PgFgaHandler) HandleDatabases() (err error) {
	return pfh.pg.CreateOrDropDatabases()
}

func (pfh PgFgaHandler) HandleRoles() (err error) {
	for roleName, roleConfig := range pfh.config.Roles {
		var options pg.RoleOptions
		for _, optionName := range roleConfig.Options {
			option := pg.RoleOption(optionName)
			if option.Valid() {
				options = append(options, option)
			}
		}
		role, err := pg.NewRole(pfh.pg, roleName, options)
		if err != nil {
			return err
		}
		for _, groupName := range roleConfig.MemberOf {
			group, err := pfh.pg.GetRole(groupName)
			if err != nil {
				return err
			}
			err = role.GrantRole(group)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
