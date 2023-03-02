package exec

import (
	"context"
	"os/exec"
)

// Interface is representing an os/exec Cmd struct.
// This makes it usable to replace for testing.
// See the `MockCommand` and `MockCommandContext` functions
// for details.
//
//go:generate mockgen -destination ../../test/mock/mockexec/cmd_mock_interface.go -package mockexec . Interface
type Interface interface {
	Run() error
	Output() ([]byte, error)
	Start() error
	Wait() error
}

var _ Interface = &exec.Cmd{}

// CommandContextFunc is the function type for the CommandContext function
type CommandContextFunc func(ctx context.Context, name string, arg ...string) Interface

// CommandContext is a wrapper for the `CommandContext()` function of the `os/exec` package.
// It is easily replaceable in unit tests with a call to `MockCommandContext()` function.
// Use this as a complete drop-in replacement for the `os/exec` package.
var CommandContext CommandContextFunc = func(ctx context.Context, name string, arg ...string) Interface {
	return exec.CommandContext(ctx, name, arg...)
}

// CommandFunc is the function type for the Command function
type CommandFunc func(name string, arg ...string) Interface

// Command is a wrapper for the `Command()` function of the `os/exec` package.
// It is easily replaceable in unit tests with a call to `MockCommandContext()` function.
// Use this as a complete drop-in replacement for the `os/exec` package.
var Command CommandFunc = func(name string, arg ...string) Interface {
	return exec.Command(name, arg...)
}
