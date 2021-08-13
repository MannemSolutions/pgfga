package ldap

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
)

type Credential struct {
	Value  string `yaml:"value"`
	File   string `yaml:"file"`
	Base64 bool   `yaml:"base64"`
}

func isExecutable(filename string) (isExecutable bool, err error) {
	fi, err := os.Lstat(filename)
	if err != nil {
		return false, err
	}
	mode := fi.Mode()
	return mode&0111 == 0111, nil
}

func fromExecutable(filename string) (value string, err error) {
	// The intent is to give an option to use a 3rd party tool to retrieve a password.
	// Or a script to hash / unhash anyway you like
	// As such running an arbitrary command set as a parameter is sot of the point.
	// #nosec
	out, err := exec.Command(filename).Output()
	if err != nil {
		return "", nil
	}
	return string(out), nil
}

func fromFile(filename string) (value string, err error) {
	isExec, err := isExecutable(filename)
	if err != nil {
		return "", err
	}
	if isExec {
		return fromExecutable(filename)
	}
	// The intent is to give an option to retrieve a password from a file.
	// As such opening a file which name is set by a variable is sort of the point.
	// #nosec
	file, err := os.Open(filename) // For read access.
	if err != nil {
		return "", err
	}
	data := make([]byte, 100)
	count, err := file.Read(data)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("file %s is empty", filename)
	}
	return string(data[:]), nil
}

func (c *Credential) GetCred() (value string, err error) {
	if c.Value != "" {
	} else if c.File != "" {
		c.Value, err = fromFile(c.File)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("either value or file must be set in a credential")
	}
	if c.Base64 {
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", err
		}
		c.Value = string(data)
		c.Base64 = false
	}
	if c.Value != "" {
		return c.Value, nil
	}
	return "", fmt.Errorf("credentials file is empty")
}
