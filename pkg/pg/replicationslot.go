package pg

type ReplicationSlots map[string]ReplicationSlot

type ReplicationSlot struct {
	handler *Handler
	name    string
	State   State `yaml:"state"`
}

func NewSlot(handler *Handler, name string) (rs *ReplicationSlot) {
	if rs, exists := handler.slots[name]; exists {
		return &rs
	}
	rs = &ReplicationSlot{
		handler: handler,
		name:    name,
		State:   Present,
	}
	handler.slots[name] = *rs
	return rs
}

func (rs ReplicationSlot) Drop() (err error) {
	ph := rs.handler
	if ph.strictOptions.Slots {
		log.Infof("skipping drop of replication slot %s (not running with strict option for slots", rs.name)
		return nil
	}
	exists, err := ph.conn.runQueryExists("SELECT slot_name FROM pg_replication_slots WHERE slot_name = $1", rs.name)
	if err != nil {
		return err
	}
	if exists {
		return ph.conn.runQueryExec("SELECT pg_drop_physical_replication_slot($1)", rs.name)
	}
	return nil
}

func (rs ReplicationSlot) Create() (err error) {
	conn := rs.handler.conn

	exists, err := conn.runQueryExists("SELECT slot_name FROM pg_replication_slots WHERE slot_name = $1", rs.name)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec("SELECT pg_create_physical_replication_slot($1)", rs.name)
		if err != nil {
			return err
		}
		log.Infof("Created replication slot '%s'", rs.name)
	}
	return nil
}
