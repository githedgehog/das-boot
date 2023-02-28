package mockexec

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.githedgehog.com/dasboot/pkg/exec"

	"github.com/golang/mock/gomock"
)

// TestCmds is holding multiple commands. This is being used for
// test suites that are running multiple commands. The commands
// are being requested in order as they are registered.
type TestCmds struct {
	cmds            []exec.CommandFunc
	cmdsWithContext []exec.CommandContextFunc
	i               int
	ic              int
}

// NewTestCmds creates a new TestCmds objects registering `cmds`
// as commands which must be executed
func NewMockCommands(cmds []exec.CommandFunc) *TestCmds {
	return &TestCmds{
		cmds: cmds,
	}
}

// AddCommands adds commands to the end of the already registered commands.
func (c *TestCmds) AddCommands(cmds ...exec.CommandFunc) {
	c.cmds = append(c.cmds, cmds...)
}

// AddCommandContexts adds commands with context to the end of the already registered commands.
func (c *TestCmds) AddCommandContexts(cmds ...exec.CommandContextFunc) {
	c.cmdsWithContext = append(c.cmdsWithContext, cmds...)
}

// Finish will make sure that all commands ran, and will panic otherwise.
// Use this at the end of a test to confirm that all commands were called.
func (c *TestCmds) Finish() {
	if c.i != len(c.cmds) {
		panic(fmt.Errorf("TestCmds: still %d mock Command()s to run", len(c.cmds)-c.i))
	}
	if c.ic != len(c.cmdsWithContext) {
		panic(fmt.Errorf("TestCmds: still %d mock CommandContext()s to run", len(c.cmdsWithContext)-c.ic))
	}
}

// Command returns the next command in the registered list of commands
// This will panic if there are no further commands registered
// (which is totally acceptable in a unit test).
func (c *TestCmds) Command() exec.CommandFunc {
	return func(name string, arg ...string) exec.Interface {
		if c.i >= len(c.cmds) {
			panic(fmt.Errorf("unregistered command trying to run: '%s %s'", name, strings.Join(arg, " ")))
		}
		defer func() { c.i += 1 }()
		return c.cmds[c.i](name, arg...)
	}
}

// CommandContext returns the next command in the registered list of commands with context.
// This will panic if there are no further commands registered
// (which is totally acceptable in a unit test).
func (c *TestCmds) CommandContext() exec.CommandContextFunc {
	return func(ctx context.Context, name string, arg ...string) exec.Interface {
		if c.ic >= len(c.cmdsWithContext) {
			panic(fmt.Errorf("unregistered command with context trying to run: '%s %s'", name, strings.Join(arg, " ")))
		}
		defer func() { c.ic += 1 }()
		return c.cmdsWithContext[c.ic](ctx, name, arg...)
	}
}

// TestCmd encapsulates a command for mocked execution in a similar fashion how
// the `*exec.Cmd` encapsulates a command for the `Command()`/`CommandContext()`
// functions of the `os/exec` package.
type TestCmd struct {
	*MockInterface
	name            string
	arg             []string
	expectedNameArg []string
}

// IsExpectedCommand will return an error if the command which is being executed
// does not match the expected command. Use this insided of the `mockFunc` in the
// `MockCommand()` or `MockCommandContext` functions to additionally test if the
// command is as expected.
func (c *TestCmd) IsExpectedCommand() error {
	if len(c.expectedNameArg) > 0 {
		nameArg := []string{c.name}
		nameArg = append(nameArg, c.arg...)
		if !reflect.DeepEqual(nameArg, c.expectedNameArg) {
			return fmt.Errorf("not expected command: '%#v', actual '%#v'", c.expectedNameArg, nameArg)
		}
	}
	return nil
}

// MockCommand returns a mocked command which can be substituted for a call to `Command()` of the `os/exec` package.
// It will replace the actual command with a mock. You can set the right expectations to its calls to `Run()` or
// `Output()` within the provided `mockFunc`. It is a good idea to also call the `IsExpectedCommand()` function to
// ensure that the function was called as expected.
//
// NOTE: If your test is going to call multiple tests, use `TestCmds` to register all commands that are going to be
// executed.
func MockCommand(t *testing.T, ctrl *gomock.Controller, expectedNameArg []string, mockFunc func(*TestCmd)) exec.CommandFunc {
	return func(name string, arg ...string) exec.Interface {
		cmd := NewMockInterface(ctrl)
		testCmd := &TestCmd{
			MockInterface:   cmd,
			name:            name,
			arg:             arg,
			expectedNameArg: expectedNameArg,
		}
		mockFunc(testCmd)
		return testCmd
	}
}

// MockCommandContext returns a mocked command which can be substituted for a call to `CommandContext()` of the `os/exec` package.
// It will replace the actual command with a mock. You can set the right expectations to its calls to `Run()` or
// `Output()` within the provided `mockFunc`. It is a good idea to also call the `IsExpectedCommand()` function to
// ensure that the function was called as expected.
//
// NOTE: If your test is going to call multiple tests, use `TestCmds` to register all commands that are going to be
// executed.
func MockCommandContext(t *testing.T, ctrl *gomock.Controller, expectedNameArg []string, mockFunc func(*TestCmd)) exec.CommandContextFunc {
	return func(_ context.Context, name string, arg ...string) exec.Interface {
		cmd := NewMockInterface(ctrl)
		testCmd := &TestCmd{
			MockInterface:   cmd,
			name:            name,
			arg:             arg,
			expectedNameArg: expectedNameArg,
		}
		mockFunc(testCmd)
		return testCmd
	}
}
