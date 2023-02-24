package partitions

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
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
			return fmt.Errorf("not expected command: '%#v', actual '%#v'", c.expectedNameArg, nameArg)
		}
	}
	return nil
}

func Test_execCommand(t *testing.T) {
	// just to make sure that without switching out execCommand, it really calls a command
	t.Run("run", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		out, err := execCommand("pwd").Output()
		if err != nil {
			panic(err)
		}
		got := strings.TrimSpace(string(out))
		if wd != got {
			t.Errorf("working directories did not match: got = %s, want = %s", got, wd)
		}
	})
}
