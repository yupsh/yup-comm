package main

import (
	"context"
	"errors"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying the wrapper's flags and
// returns the parsed accessor.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  spec.Flags,
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func TestColumnDigits(t *testing.T) {
	cases := []struct {
		tok  token
		want [4]bool
		ok   bool
	}{
		{"-1", [4]bool{1: true}, true},
		{"-12", [4]bool{1: true, 2: true}, true},
		{"-123", [4]bool{1: true, 2: true, 3: true}, true},
		{"a.txt", [4]bool{}, false},
		{"-", [4]bool{}, false},
		{"-4", [4]bool{}, false},
		{"-1x", [4]bool{}, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.tok), func(t *testing.T) {
			cols, ok := columnDigits(tc.tok)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if [4]bool(cols) != tc.want {
				t.Fatalf("cols=%v, want %v", cols, tc.want)
			}
		})
	}
}

func TestPartition(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		opts  int
		files int
	}{
		{"filesOnly", []string{name, "a.txt", "b.txt"}, 0, 2},
		{"shortFlags", []string{name, "-1", "-2", "a.txt", "b.txt"}, 2, 2},
		{"grouped", []string{name, "-123", "a.txt", "b.txt"}, 3, 2},
		{"longFlags", []string{name, "--suppress-column-1", "a.txt", "b.txt"}, 1, 2},
		{"mergedDuplicate", []string{name, "-1", "-1", "a.txt", "b.txt"}, 1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, files := partition(parse(t, tc.args...))
			if len(opts) != tc.opts {
				t.Fatalf("opts len=%d, want %d", len(opts), tc.opts)
			}
			if len(files) != tc.files {
				t.Fatalf("files len=%d, want %d", len(files), tc.files)
			}
		})
	}
}

func TestOptions_AllColumns(t *testing.T) {
	sup := columnSuppression{1: true, 2: true, 3: true}
	if got := len(sup.options()); got != 3 {
		t.Fatalf("options len=%d, want 3", got)
	}
}

func TestBuild_Filter(t *testing.T) {
	inv := clix.Invocation{Args: parse(t, name, "a.txt", "b.txt"), Fs: afero.NewMemMapFs()}
	src, filter, err := build(inv)
	if err != nil || src == nil || filter == nil {
		t.Fatalf("build: src=%v filter=%v err=%v", src, filter, err)
	}
}

func TestBuild_OperandCount(t *testing.T) {
	inv := clix.Invocation{Args: parse(t, name, "only.txt"), Fs: afero.NewMemMapFs()}
	src, filter, err := build(inv)
	if !errors.Is(err, ErrOperandCount) {
		t.Fatalf("err=%v, want ErrOperandCount", err)
	}
	if src != nil || filter != nil {
		t.Fatalf("src=%v filter=%v, want both nil on error", src, filter)
	}
	if err.Error() != string(ErrOperandCount) {
		t.Fatalf("message=%q, want %q", err.Error(), string(ErrOperandCount))
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
