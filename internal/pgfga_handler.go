package internal

import (
	"fmt"
	"github.com/mannemsolutions/pgfga/pkg/ldap"
	"github.com/mannemsolutions/pgfga/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
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

	pfh.pg = pg.NewPgHandler(config.PgDsn, config.StrictConfig, config.DbsConfig, config.Slots)

	return pfh, nil
}

func (pfh PgFgaHandler) Handle() {
	time.Sleep(pfh.config.GeneralConfig.RunDelay)

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
	err = pfh.HandleSlots()
	if err != nil {
		log.Fatal(err)
	}
}

func (pfh PgFgaHandler) HandleUsers() (err error) {
	for userName, userConfig := range pfh.config.UserConfig {
		options := make(pg.RoleOptions)
		for _, optionName := range userConfig.Options {
			option, err := pg.NewRoleOption(optionName)
			if err != nil {
				return err
			}
			options[optionName] = option
		}
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
			baseRole, err := pg.NewRole(pfh.pg, baseGroup.Name(), options, userConfig.State)
			if err != nil {
				return err
			}
			err = baseRole.ResetPassword()
			if err != nil {
				return err
			}
			for _, ms := range baseGroup.MembershipTree() {
				_, err = pg.NewRole(pfh.pg, ms.Member.Name(), pg.LoginOptions, userConfig.State)
				if err != nil {
					return err
				}
				err = pfh.pg.GrantRole(ms.Member.Name(), baseGroup.Name())
				if err != nil {
					return err
				}
			}
		case "ldap-user", "clientcert":
			log.Debugf("Configuring user %s with %s", userName, userConfig.Auth)
			options.AddOption(pg.LoginOption)
			user, err := pg.NewRole(pfh.pg, userName, options, userConfig.State)
			if err != nil {
				return err
			}
			err = user.ResetPassword()
			if err != nil {
				return err
			}
			if userConfig.State.Bool() {
				for _, granted := range userConfig.MemberOf {
					err := pfh.pg.GrantRole(userName, granted)
					if err != nil {
						return err
					}
				}
			}
		case "password", "md5":
			options.AddOption(pg.LoginOption)
			user, err := pg.NewRole(pfh.pg, userName, options, userConfig.State)
			if err != nil {
				return err
			}
			// Note: if no password is set, it will be reset...
			err = user.SetPassword(userConfig.Password)
			if err != nil {
				return err
			}
			err = user.SetExpiry(userConfig.Expiry)
			if err != nil {
				return err
			}
		default:
			log.Fatalf("Invalid auth %s for user %s", userConfig.Auth, userName)
		}
	}
	return nil
}

func (pfh PgFgaHandler) HandleDatabases() (err error) {
	return pfh.pg.CreateOrDropDatabases()
}

func (pfh PgFgaHandler) HandleRoles() (err error) {
	for roleName, roleConfig := range pfh.config.Roles {
		options := make(pg.RoleOptions)
		for _, optionName := range roleConfig.Options {
			option, err := pg.NewRoleOption(optionName)
			if err != nil {
				return err
			}
			options[optionName] = option
		}
		role, err := pg.NewRole(pfh.pg, roleName, options, roleConfig.State)
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
func (pfh PgFgaHandler) HandleSlots() (err error) {
	return pfh.pg.CreateOrDropSlots()
}
