package pg

type Handler struct {
	conn          *Conn
	strictOptions StrictOptions
	databases     Databases
	roles         Roles
	slots         ReplicationSlots
}

func NewPgHandler(connParams Dsn, options StrictOptions, databases Databases, slots []string) (ph *Handler) {
	ph = &Handler{
		conn:          NewConn(connParams),
		strictOptions: options,
		databases:     databases,
		roles:         make(Roles),
		slots:         make(ReplicationSlots),
	}
	for _, slotName := range slots {
		slot := NewSlot(ph, slotName)
		ph.slots[slotName] = *slot
	}
	ph.setDefaults()
	return ph
}

func (ph *Handler) setDefaults() {
	for name, db := range ph.databases {
		db.handler = ph
		db.name = name
		db.SetDefaults()
	}
	for name, rs := range ph.slots {
		rs.handler = ph
		rs.name = name
	}
}

func (ph *Handler) GetDb(dbName string) (d *Database) {
	// NewDatabase does everything we need to do
	return NewDatabase(ph, dbName, "")
}

func (ph *Handler) GetRole(roleName string) (d *Role, err error) {
	// NewDatabase does everything we need to do
	return NewRole(ph, roleName, RoleOptions{})
}

func (ph *Handler) GrantRole(granteeName string, grantedName string) (err error) {
	// NewDatabase does everything we need to do
	grantee, err := ph.GetRole(granteeName)
	if err != nil {
		return err
	}
	granted, err := ph.GetRole(grantedName)
	if err != nil {
		return err
	}
	return grantee.GrantRole(granted)
}
func (ph *Handler) CreateOrDropDatabases() (err error) {
	for _, d := range ph.databases {
		if d.State.value {
			err = d.Create()
		} else {
			err = d.Drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (ph *Handler) CreateOrDropSlots() (err error) {
	for _, d := range ph.slots {
		if d.State.value {
			err = d.Create()
		} else {
			err = d.Drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (ph *Handler) StrictifyRoles() (err error) {
	return nil
}

func (ph *Handler) StrictifyDatabases() (err error) {
	return nil
}
func (ph *Handler) StrictifyExtensions() (err error) {
	return nil
}
