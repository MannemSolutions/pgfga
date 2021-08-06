package ldap

import (
	"fmt"
	"github.com/go-ldap/ldap/v3"
)


type Handler struct {
	servers    []string
	userName   string
	password   string
	conn       *ldap.Conn
	maxRetries int
	members    Members
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
		members: make(Members),
	}
}

func (lh *Handler) Connect() (err error){
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

func (lh Handler) GetMembers(baseDN string, filter string) (baseGroup *Member, err error) {
	err = lh.Connect()
	if err != nil {
		return nil, err
	}
	baseGroup, err = lh.members.GetById(baseDN, true)
	if err != nil {
		return nil, err
	}
	searchRequest := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ldap.DerefAlways, 0, 0, false,
		filter, []string{"dn", "cn", "memberUid"}, nil)
	sr, err := lh.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	for _, entry := range sr.Entries {
		group, err := lh.members.GetById(entry.DN, true)
		if err != nil {
			return nil, err
		}
		group.AddParent(baseGroup)
		for _, memberUid := range entry.GetAttributeValues("memberUid") {
			member, err := lh.members.GetById(memberUid, true)
			if err != nil {
				return nil, err
			}
			member.AddParent(group)
			member.SetMType(UserMType)
			log.Debugf("%s: %v", member.Name(), group.Name())
		}
	}
	return baseGroup, nil
}