package ldap

type Config struct {
	Usr        Credential `yaml:"user"`
	Pwd        Credential `yaml:"password"`
	Servers    []string   `yaml:"servers"`
	MaxRetries int        `yaml:"conn_retries"`
}

func (c *Config) SetDefaults() {
	if c.MaxRetries < 1 {
		c.MaxRetries = 1
	}
}

func (c Config) User() (user string, err error) {
	user, err = c.Usr.GetCred()
	if err != nil {
		return "", err
	}
	return user, nil
}

func (c Config) Password() (pwd string, err error) {
	pwd, err = c.Pwd.GetCred()
	if err != nil {
		return "", err
	}
	return pwd, nil
}
