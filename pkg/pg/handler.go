package pg

type Handler struct {
	conn          *Conn
	strictOptions StrictOptions
	databases     Databases
	roles         Roles
}

func NewPgHandler(connParams Dsn, options StrictOptions, databases Databases) (ph *Handler) {
	ph = &Handler{
		conn:          NewConn(connParams),
		strictOptions: options,
		databases:     databases,
		roles:         make(Roles),
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

func (ph *Handler) StrictifyRoles() (err error) {
	return nil
}

func (ph *Handler) StrictifyDatabases() (err error) {
	return nil
}
func (ph *Handler) StrictifyExtensions() (err error) {
	return nil
}
