package ldap

import (
	"errors"
	"regexp"
	"strings"
)

type MemberType int

const (
	 GroupMType MemberType = iota
	 UserMType
	 UnknownMType
)

type Member struct {
	dn       string
	pair     string
	name     string
	mType	 MemberType
	parents  Members
	children Members
}

func validDn(dn string) bool {
	var validDn = regexp.MustCompile(`^([a-zA-Z]+=[a-zA-Z0-9]+,)*[a-zA-Z]+=[a-zA-Z0-9]+$`)
	return validDn.MatchString(dn)
}

func validLdapPair(pair string) (isValid bool){
	var validPair = regexp.MustCompile(`^[a-zA-Z]+=[a-zA-Z0-9]+$`)
	return validPair.MatchString(pair)
}

func NewMember(Id string) (m *Member, err error) {
	m = &Member{
		parents: make(Members),
		children: make(Members),
	}
	return m, m.SetFromId(Id)
}

func GetMemberType(key string) (mt MemberType){
	switch key {
	case "cn":
		return GroupMType
	case "uid":
		return UserMType
	default:
		return UnknownMType
	}
}

// SetFromId allows dn, id, and name to be set if they are not set yet, but determines it makes sense before doing so
func (m *Member) SetFromId(Id string) (err error) {
	if m.dn != "" {
		return nil
	}
	if validDn(Id) {
		pair := strings.Split(Id, ",")[0]
		if m.pair != "" && m.pair != pair {
			return errors.New("trying to set dn, while pair is already set differently")
		}
		key := strings.Split(pair, "=")[0]
		name := strings.Split(pair, "=")[1]
		if m.name != "" && m.name != name {
			return errors.New("trying to set dn, while name is already set differently")
		}
		m.dn = Id
		m.pair = pair
		m.name = name
		m.mType = GetMemberType(key)
		return
	}
	if m.pair != "" {
		return nil
	}
	if validLdapPair(Id) {
		key := strings.Split(Id, "=")[0]
		name := strings.Split(Id, "=")[1]
		if m.name != "" && m.name != name {
			return errors.New("trying to set pai, while name is already set differently")
		}
		m.pair = Id
		m.name = name
		m.mType = GetMemberType(key)
	}
	if m.name != "" {
		return nil
	}
	m.name = Id
	m.mType = UnknownMType
	return nil
}

func (m *Member) SetMType(mt MemberType) (err error){
	if m.mType != UnknownMType {
		return errors.New("cannot set MemberType when already set")
	}
	m.mType = mt
	return nil
}

func (m *Member) Name() (name string) {
	return m.name
}

func (m *Member) Pair() (pair string) {
	return m.pair
}

func (m *Member) Dn() (dn string) {
	return m.dn
}

func (m *Member) AddParent(p *Member) {
	if m.dn == p.dn {
		// This is me, myself and I. Skipping.
		return
	}
	if _, exists := m.parents[p.name]; exists {
		// Already exists, so just return that one
		return
	}
	m.parents[p.name] = p
	p.children[m.name] = m
}

type Membership struct {
	Member   *Member
	MemberOf *Member
}

type Memberships []Membership

func (m *Member) MembershipTree() (mss Memberships) {
	for _, member := range m.children {
		ms := Membership{
			Member: member,
			MemberOf: m,
		}
		mss = append(mss, ms)
		subMss := member.MembershipTree()
		mss = append(mss, subMss...)
	}
	return mss
}

type Members map[string]*Member

func (ms Members) GetById(Id string, AddWhenMissing bool) (m *Member, err error){
	m, err = NewMember(Id)
	if err != nil {
		return m, err
	}
	if _, exists := ms[m.name]; exists {
		// Already exists, so just return that one
		return ms[m.name], nil
	}
	if ! AddWhenMissing {
		return &Member{}, nil
	}
	// ms is not a *Members, cause Members is already a points (map[string]Member).
	//So check if after leaving this method, that ms actually still holds the new values
	ms[m.name] = m
	ms[m.pair] = m
	ms[m.dn] = m
	return m, nil
}