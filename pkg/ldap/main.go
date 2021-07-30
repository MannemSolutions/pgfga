package ldap

import (
	"fmt"
	"github.com/go-ldap/ldap/v3"
)

type LdapHandler struct {
	servers    []string
	userName   string
	password   string
	conn       *ldap.Conn
	maxRetries int
}

func NewLdapHandler(servers []string, user string, password string, maxRetries int) (lh *LdapHandler) {
	if maxRetries < 1 {
		maxRetries = 1
	}
	return &LdapHandler{
		servers: servers,
		userName: user,
		password: password,
		maxRetries: maxRetries,
	}
}

func (lh LdapHandler) Connect() (err error){
	if lh.conn != nil {
		return nil
	}
	for i:= 0; i < lh.maxRetries; i++ {
		for _, server := range lh.servers {
			conn, err := ldap.DialURL(server)
			if err != nil {
				continue
			}
			err = conn.Bind(lh.userName,lh.password)
			if err != nil {
				return err
			}
			lh.conn = conn
			return nil
		}
	}
	return fmt.Errorf("none of the ldap servers are available")
}

type LdapMembership struct {
	Member   string
	MemberOf string
}

func (lh LdapHandler) GetMemberships(baseDN string, filter string) (ms []LdapMembership, err error) {
	err = lh.Connect()
	if err != nil {
		return ms, err
	}
	searchRequest := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ldap.DerefAlways, 0, 0, false,
		filter, []string{"dn", "cn"}, nil)
	sr, err := lh.conn.Search(searchRequest)
	if err != nil {
		return ms, err
	}

	for _, entry := range sr.Entries {
		ms = append(ms, LdapMembership{Member: entry.DN, MemberOf: entry.GetAttributeValue("cn")})
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))
	}
	return ms, nil
}