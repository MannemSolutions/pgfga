package internal

import (
	"fmt"
	"github.com/mannemsolutions/pgfga/pkg/ldap"
	"github.com/mannemsolutions/pgfga/pkg/pg"
)

type PgFgaHandler struct {
	config FgaConfig
	pg     *pg.PgHandler
	ldap   *ldap.LdapHandler
}

func NewPgFgaHandler() (pfh *PgFgaHandler, err error) {
	config, err := NewConfig()
	if err != nil {
		return pfh, err
	}

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

	pfh.pg = pg.NewPgHandler(config.PgConfig.DSN())

	return pfh, nil
}

func (pfh PgFgaHandler) Handle() (err error) {
	err = pfh.HandleUsers()
	if err != nil {
		return err
	}
	return nil
}

func (pfh PgFgaHandler) HandleUsers() (err error) {
	for userName, userConfig := range pfh.config.UserConfig {
		switch userConfig.Auth {
		case "ldap-group":
			fmt.Sprintf("Configuring role from ldap for %s", userName)
			if userConfig.BaseDN == "" || userConfig.Filter == "" {
				return fmt.Errorf("ldapbasedn and ldapfilter must be set for %s (auth: 'ldap-group')", userName)
			}
			mss, err := pfh.ldap.GetMemberships(userConfig.BaseDN, userConfig.Filter)
			if err != nil {
				return err
			}
			for _, ms := range mss {
				pfh.pg.GrantRole(ms.Member, ms.MemberOf)
			}
		}
	}
	return nil
}
