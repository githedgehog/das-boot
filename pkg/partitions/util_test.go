package partitions

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

type symwalkFn struct {
	paths []string
}

func (s *symwalkFn) walkDirFunc(path string, d fs.DirEntry, err error) error {
	s.paths = append(s.paths, path)
	return nil
}

func (s *symwalkFn) walkedPaths() []string {
	ret := make([]string, len(s.paths))
	copy(ret, s.paths)
	sort.Strings(ret)
	return ret
}

func TestWalkDir(t *testing.T) {

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	base := filepath.Join(pwd, "testdata", "WalkDir")

	errFilepathRelFailed := errors.New("filepath.Rel failed")
	errFilepathEvalSymlinksFailed := errors.New("filepath.EvalSymlinks failed")

	type args struct {
		path       string
		maxLevel   uint
		exclusions []string
	}
	tests := []struct {
		name                 string
		args                 args
		wantErr              bool
		wantErrToBe          error
		wantPaths            []string
		filepathRel          func(basepath string, targpath string) (string, error)
		filepathEvalSymlinks func(path string) (string, error)
		osLstat              func(name string) (fs.FileInfo, error)
	}{
		{
			name: "no subfolder",
			args: args{
				path:       filepath.Join(base, "flat"),
				maxLevel:   1,
				exclusions: []string{},
			},
			wantErr: false,
			wantPaths: []string{
				filepath.Join(base, "flat"),
				filepath.Join(base, "flat", "no-subfolder"),
			},
		},
		{
			name: "no subfolder with exclusions",
			args: args{
				path:       filepath.Join(base, "flat2"),
				maxLevel:   1,
				exclusions: []string{"exclude-symlink"},
			},
			wantErr: false,
			wantPaths: []string{
				filepath.Join(base, "flat2"),
				filepath.Join(base, "flat2", "exclude-symlink"),
				filepath.Join(base, "flat2", "no-subfolder"),
			},
		},
		{
			name:        "filepath rel fails",
			wantErr:     true,
			wantErrToBe: errFilepathRelFailed,
			filepathRel: func(basepath, targpath string) (string, error) {
				return "", errFilepathRelFailed
			},
			wantPaths: []string{},
		},
		{
			name: "resolving one level of symlinks",
			args: args{
				path:       filepath.Join(base, "has-symlinks"),
				maxLevel:   1,
				exclusions: []string{},
			},
			wantErr: false,
			wantPaths: []string{
				filepath.Join(base, "has-symlinks"),
				filepath.Join(base, "has-symlinks", "flat"),
				filepath.Join(base, "has-symlinks", "flat", "no-subfolder"),
			},
		},
		{
			name: "filepath evalsymlinks failed",
			args: args{
				path:       filepath.Join(base, "has-symlinks"),
				maxLevel:   1,
				exclusions: []string{},
			},
			wantErr:     true,
			wantErrToBe: errFilepathEvalSymlinksFailed,
			filepathEvalSymlinks: func(path string) (string, error) {
				return "", errFilepathEvalSymlinksFailed
			},
			wantPaths: []string{
				filepath.Join(base, "has-symlinks"),
			},
		},
		{
			name: "resolving two level of symlinks",
			args: args{
				path:       filepath.Join(base, "has-nested-symlinks"),
				maxLevel:   2,
				exclusions: []string{},
			},
			wantErr: false,
			wantPaths: []string{
				filepath.Join(base, "has-nested-symlinks"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks", "flat"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks", "flat", "no-subfolder"),
			},
		},
		{
			name: "abort level of symlinks after one level",
			args: args{
				path:       filepath.Join(base, "has-nested-symlinks"),
				maxLevel:   1,
				exclusions: []string{},
			},
			wantErr: false,
			wantPaths: []string{
				filepath.Join(base, "has-nested-symlinks"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks", "flat"),
			},
		},
		{
			name: "abort level of symlinks after one level with os lstat failing",
			args: args{
				path:       filepath.Join(base, "has-nested-symlinks"),
				maxLevel:   1,
				exclusions: []string{},
			},
			wantErr: false,
			osLstat: func(name string) (fs.FileInfo, error) {
				return nil, fmt.Errorf("os lstat failed")
			},
			wantPaths: []string{
				filepath.Join(base, "has-nested-symlinks"),
				filepath.Join(base, "has-nested-symlinks", "has-symlinks"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filepathRel != nil {
				oldFilepathRel := filepathRel
				defer func() {
					filepathRel = oldFilepathRel
				}()
				filepathRel = tt.filepathRel
			}
			if tt.filepathEvalSymlinks != nil {
				oldFilepathEvalSymlinks := filepathEvalSymlinks
				defer func() {
					filepathEvalSymlinks = oldFilepathEvalSymlinks
				}()
				filepathEvalSymlinks = tt.filepathEvalSymlinks
			}
			if tt.osLstat != nil {
				oldOsLstat := osLstat
				defer func() {
					osLstat = oldOsLstat
				}()
				osLstat = tt.osLstat
			}
			s := &symwalkFn{}
			if err := WalkDir(tt.args.path, s.walkDirFunc, tt.args.maxLevel, tt.args.exclusions...); (err != nil) != tt.wantErr {
				t.Errorf("WalkDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("WalkDir() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			walkedPaths := s.walkedPaths()
			if !reflect.DeepEqual(walkedPaths, tt.wantPaths) {
				t.Errorf("WalkDir() paths = %#v, wantPaths = %#v", walkedPaths, tt.wantPaths)
				return
			}
		})
	}
}

func Test_walkDir(t *testing.T) {
	type args struct {
		filename    string
		linkDirname string
		walkFn      fs.WalkDirFunc
		maxLevel    uint
		exclusions  []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := walkDir(tt.args.filename, tt.args.linkDirname, tt.args.walkFn, tt.args.maxLevel, tt.args.exclusions); (err != nil) != tt.wantErr {
				t.Errorf("walkDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
