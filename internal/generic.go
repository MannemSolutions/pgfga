package internal

import (
	"encoding/json"
	"fmt"
)

// Use this function to print a human readable version of the returned struct.
func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}