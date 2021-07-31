package ldap

import (
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger
func Initialize(sugar *zap.SugaredLogger) {
	log = sugar
}

type Handler struct {
	servers    []string
	userName   string
	password   string
	conn       *ldap.Conn
	maxRetries int
}

func NewLdapHandler(servers []string, user string, password string, maxRetries int) (lh *Handler) {
	if maxRetries < 1 {
		maxRetries = 1
	}
	return &Handler{
		servers: servers,
		userName: user,
		password: password,
		maxRetries: maxRetries,
	}
}

func (lh Handler) Connect() (err error){
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

type Membership struct {
	Member   string
	MemberOf string
}

func (lh Handler) GetMemberships(baseDN string, filter string) (ms []Membership, err error) {
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
		ms = append(ms, Membership{Member: entry.DN, MemberOf: entry.GetAttributeValue("cn")})
		log.Debugf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))
	}
	return ms, nil
}