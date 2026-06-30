package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	command "github.com/gloo-foo/cmd-comm"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const (
	flagSuppressColumn1 = "suppress-column-1"
	flagSuppressColumn2 = "suppress-column-2"
	flagSuppressColumn3 = "suppress-column-3"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `comm [OPTIONS] FILE1 FILE2

Compare sorted files FILE1 and FILE2 line by line.

With no options, produce three-column output. Column one contains
lines unique to FILE1, column two contains lines unique to FILE2,
and column three contains lines common to both files.`

// Error is the sentinel error type for this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrOperandCount is raised when comm is not given exactly two file operands.
const ErrOperandCount Error = "comm takes exactly two FILE operands"

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags (e.g. grep -v) while
// still exposing the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the comm CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, _ io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), translateColumnFlags(args)); err != nil {
		_, _ = fmt.Fprintf(stderr, "comm: %v\n", err)
		return 1
	}
	return 0
}

// columnFlag maps a comm suppression digit (1/2/3) to its long flag name. Only
// these three digits are GNU comm column-suppression flags.
var columnFlag = map[rune]string{
	'1': "--" + flagSuppressColumn1,
	'2': "--" + flagSuppressColumn2,
	'3': "--" + flagSuppressColumn3,
}

// translateColumnFlags rewrites GNU comm's digit-leading short flags (-1, -2,
// -3, and grouped forms like -12 or -123) into their long-flag equivalents.
// urfave/cli/v3's parser treats a leading-digit token (e.g. "-1") as a
// positional rather than a short flag, so this preserves GNU comm parity.
// Tokens after a bare "--" terminator are left untouched so filenames are safe.
func translateColumnFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			return append(out, args[i:]...)
		}
		out = append(out, expandColumnFlag(arg)...)
	}
	return out
}

// expandColumnFlag returns the long flags for a column-suppression token, or the
// token unchanged when it is not one (any non-digit rune leaves it untouched).
func expandColumnFlag(arg string) []string {
	digits, ok := strings.CutPrefix(arg, "-")
	if !ok || digits == "" {
		return []string{arg}
	}
	long := make([]string, 0, len(digits))
	for _, r := range digits {
		name, isColumn := columnFlag[r]
		if !isColumn {
			return []string{arg}
		}
		long = append(long, name)
	}
	return long
}

func newApp(version string, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "comm",
		Version:         version,
		Usage:           "compare two sorted files line by line",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    flagSuppressColumn1,
				Aliases: []string{"1"},
				Usage:   "suppress column 1 (lines unique to FILE1)",
			},
			&cli.BoolFlag{
				Name:    flagSuppressColumn2,
				Aliases: []string{"2"},
				Usage:   "suppress column 2 (lines unique to FILE2)",
			},
			&cli.BoolFlag{
				Name:    flagSuppressColumn3,
				Aliases: []string{"3"},
				Usage:   "suppress column 3 (lines common to both files)",
			},
		},
		Action: action(stdout, fs),
	}
}

func action(stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		if c.NArg() != 2 {
			return ErrOperandCount
		}
		// Both operands are passed to Comm as File positionals: positional[0]
		// is input1, positional[1] is input2. CommFs routes their opens through
		// the injected filesystem. The pipeline source is empty because Comm
		// reads input1 from positional[0] rather than the upstream stream.
		file1, file2 := gloo.File(c.Args().Get(0)), gloo.File(c.Args().Get(1))
		opts := append([]any{file1, file2, command.CommFs(fs)}, options(c)...)
		_, err := gloo.Run(gloo.ByteFileSource(fs, nil), gloo.ByteWriteTo(stdout), command.Comm(opts...))
		return err
	}
}

func options(c *cli.Command) []any {
	var opts []any
	if c.Bool(flagSuppressColumn1) {
		opts = append(opts, command.CommSuppressColumn1)
	}
	if c.Bool(flagSuppressColumn2) {
		opts = append(opts, command.CommSuppressColumn2)
	}
	if c.Bool(flagSuppressColumn3) {
		opts = append(opts, command.CommSuppressColumn3)
	}
	return opts
}
