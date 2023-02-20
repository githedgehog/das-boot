package partitions

import (
	"fmt"
	"reflect"
	"strings"
)

type testCmd struct {
	Cmd
	name            string
	arg             []string
	expectedNameArg []string
}

func (c *testCmd) IsExpectedCommand() error {
	if len(c.expectedNameArg) > 0 {
		nameArg := []string{c.name}
		nameArg = append(nameArg, c.arg...)
		if !reflect.DeepEqual(nameArg, c.expectedNameArg) {
			return fmt.Errorf("not expected command: '%s', actual '%s'", strings.Join(c.expectedNameArg, " "), strings.Join(nameArg, " "))
		}
	}
	return nil
}
