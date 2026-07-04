// Command yup-comm is the CLI wrapper around github.com/gloo-foo/cmd-comm.
package main

import (
	"strings"

	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-comm"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name                = "comm"
	flagSuppressColumn1 = "suppress-column-1"
	flagSuppressColumn2 = "suppress-column-2"
	flagSuppressColumn3 = "suppress-column-3"
)

// Error is the sentinel error type for this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrOperandCount is raised when comm is not given exactly two file operands.
const ErrOperandCount Error = "comm takes exactly two FILE operands"

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `comm [OPTIONS] FILE1 FILE2

Compare sorted files FILE1 and FILE2 line by line.

With no options, produce three-column output. Column one contains
lines unique to FILE1, column two contains lines unique to FILE2,
and column three contains lines common to both files.`

// spec declares the comm wrapper. comm reads both FILE operands itself, so its
// command is the whole pipeline over an empty upstream source (a filter whose
// input is unused).
var spec = clix.Spec{
	Name:     name,
	Summary:  "compare two sorted files line by line",
	Synopsis: synopsis,
	Build:    build,
	Flags: []urf.Flag{
		&urf.BoolFlag{Name: flagSuppressColumn1, Usage: "suppress column 1 (lines unique to FILE1)"},
		&urf.BoolFlag{Name: flagSuppressColumn2, Usage: "suppress column 2 (lines unique to FILE2)"},
		&urf.BoolFlag{Name: flagSuppressColumn3, Usage: "suppress column 3 (lines common to both files)"},
	},
}

// build maps the invocation to comm's pipeline: the two FILE operands are read
// through the injected filesystem as comm's positionals, and the suppression
// options hide the requested columns. Anything other than exactly two FILE
// operands is a usage error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	opts, files := partition(inv.Args)
	if len(files) != 2 {
		return nil, nil, ErrOperandCount
	}
	opts = append(opts, clix.File(files[0]), clix.File(files[1]), command.CommFs(inv.Fs))
	return clix.Files(inv.Fs), command.Comm(opts...), nil
}

// token is one raw command-line operand, which may be a digit-leading
// column-suppression flag (e.g. -1 or -12) or a file path.
type token string

// columnSuppression records which of comm's three columns are hidden; only
// indexes 1, 2, and 3 are used.
type columnSuppression [4]bool

// partition splits comm's operands into the suppression options and the FILE
// operands. Column suppression comes from either the long --suppress-column-N
// flags or GNU comm's digit-leading short flags (-1, -2, -3 and grouped forms
// like -12), which urfave/cli treats as positionals; both are honored.
func partition(c *urf.Command) ([]any, []string) {
	seen := columnsFromFlags(c)
	var files []string
	for _, tok := range c.Args().Slice() {
		if cols, ok := columnDigits(token(tok)); ok {
			seen = seen.merge(cols)
			continue
		}
		files = append(files, tok)
	}
	return seen.options(), files
}

// columnsFromFlags reads the three long suppression flags into a suppression
// set.
func columnsFromFlags(c *urf.Command) columnSuppression {
	return columnSuppression{
		1: c.Bool(flagSuppressColumn1),
		2: c.Bool(flagSuppressColumn2),
		3: c.Bool(flagSuppressColumn3),
	}
}

// columnDigits parses a digit-leading suppression token (-1, -2, -3, or grouped
// forms like -12) into the columns it hides. The bool reports whether the token
// was such a flag; any non-digit content leaves it to be treated as a file.
func columnDigits(tok token) (columnSuppression, bool) {
	digits, ok := strings.CutPrefix(string(tok), "-")
	if !ok || digits == "" {
		return columnSuppression{}, false
	}
	var cols columnSuppression
	for _, r := range digits {
		if r < '1' || r > '3' {
			return columnSuppression{}, false
		}
		cols[r-'0'] = true
	}
	return cols, true
}

// merge returns the union of two suppression sets.
func (c columnSuppression) merge(o columnSuppression) columnSuppression {
	for i := 1; i <= 3; i++ {
		c[i] = c[i] || o[i]
	}
	return c
}

// options folds the suppression set into comm's option values.
func (c columnSuppression) options() []any {
	var opts []any
	if c[1] {
		opts = append(opts, command.CommSuppressColumn1)
	}
	if c[2] {
		opts = append(opts, command.CommSuppressColumn2)
	}
	if c[3] {
		opts = append(opts, command.CommSuppressColumn3)
	}
	return opts
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
