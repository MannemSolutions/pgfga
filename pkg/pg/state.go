package pg

import (
	"fmt"
	"strings"
)

const (
	statePresent = iota
	stateAbsent
)

// State represents the state of a pg object (Present or Absent)
type State struct {
	value int
}

var (
	// Present means the object should be created
	Present = State{statePresent}
	// Absent means the object should be removed
	Absent = State{stateAbsent}

	toState = map[string]State{
		"present": Present,
		"absent":  Absent,
		"":        Present,
	}
)

func (s State) String() string {
	if s.value == stateAbsent {
		return "Absent"
	}
	return "Present"
}

func (s State) Bool() bool {
	return s.value != stateAbsent
}

// MarshalYAML marshals the enum as a quoted json string
func (s State) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML converts a yaml string to the enum value
func (s *State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	str = strings.ToLower(str)
	if state, exists := toState[str]; exists {
		s.value = state.value
		return nil
	}
	return fmt.Errorf("invalid state %s (should be Present or Absent)", str)
}
