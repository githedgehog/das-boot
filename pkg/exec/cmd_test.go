package exec

import (
	"context"
	"os"
	"strings"
	"testing"
)

func Test_Command(t *testing.T) {
	// just to make sure that without switching out execCommand, it really calls a command
	t.Run("run", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		out, err := Command("pwd").Output()
		if err != nil {
			panic(err)
		}
		got := strings.TrimSpace(string(out))
		if wd != got {
			t.Errorf("working directories did not match: got = %s, want = %s", got, wd)
		}
	})
}

func Test_CommandContext(t *testing.T) {
	// just to make sure that without switching out execCommand, it really calls a command
	t.Run("run", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		out, err := CommandContext(context.Background(), "pwd").Output()
		if err != nil {
			panic(err)
		}
		got := strings.TrimSpace(string(out))
		if wd != got {
			t.Errorf("working directories did not match: got = %s, want = %s", got, wd)
		}
	})
}
