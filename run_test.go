package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	const (
		file1 = "/a.txt"
		file2 = "/b.txt"
	)
	files := map[string]string{
		file1: "apple\nbanana\ncherry\n",
		file2: "banana\ncherry\nkiwi\n",
	}

	cases := []struct {
		files      map[string]string
		name       string
		version    string
		wantOut    string
		wantErrSub string
		args       []string
		wantCode   int
	}{
		{
			name:    "default three columns",
			args:    []string{"comm", file1, file2},
			files:   files,
			wantOut: "apple\n\t\tbanana\n\t\tcherry\n\tkiwi\n",
		},
		{
			name:    "suppress column 1",
			args:    []string{"comm", "-1", file1, file2},
			files:   files,
			wantOut: "\tbanana\n\tcherry\nkiwi\n",
		},
		{
			name:    "suppress column 2",
			args:    []string{"comm", "-2", file1, file2},
			files:   files,
			wantOut: "apple\n\tbanana\n\tcherry\n",
		},
		{
			name:    "suppress column 3",
			args:    []string{"comm", "-3", file1, file2},
			files:   files,
			wantOut: "apple\n\tkiwi\n",
		},
		{
			name:    "grouped suppress columns 1 and 2",
			args:    []string{"comm", "-12", file1, file2},
			files:   files,
			wantOut: "banana\ncherry\n",
		},
		{
			name:    "grouped suppress columns 2 and 3",
			args:    []string{"comm", "-23", file1, file2},
			files:   files,
			wantOut: "apple\n",
		},
		{
			name:    "grouped suppress all columns",
			args:    []string{"comm", "-123", file1, file2},
			files:   files,
			wantOut: "",
		},
		{
			name:    "separate suppress columns 1 and 2",
			args:    []string{"comm", "-1", "-2", file1, file2},
			files:   files,
			wantOut: "banana\ncherry\n",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"comm", "--version"},
			wantOut: "comm version 1.2.3\n",
		},
		{
			name:       "missing operand",
			args:       []string{"comm", file1},
			files:      files,
			wantCode:   1,
			wantErrSub: "comm: comm takes exactly two FILE operands",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"comm", "--nope"},
			wantCode:   1,
			wantErrSub: "comm:",
		},
		{
			name:       "input2 file missing",
			args:       []string{"comm", file1, "/missing.txt"},
			files:      map[string]string{file1: files[file1]},
			wantCode:   1,
			wantErrSub: "comm:",
		},
		{
			name:       "input1 file missing",
			args:       []string{"comm", "/missing.txt", file2},
			files:      map[string]string{file2: files[file2]},
			wantCode:   1,
			wantErrSub: "comm:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for path, content := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(""), &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

func TestTranslateColumnFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "single digit flags expand to long flags",
			args: []string{"comm", "-1", "-3", "a", "b"},
			want: []string{"comm", "--suppress-column-1", "--suppress-column-3", "a", "b"},
		},
		{
			name: "grouped digits expand to multiple long flags",
			args: []string{"comm", "-123", "a", "b"},
			want: []string{"comm", "--suppress-column-1", "--suppress-column-2", "--suppress-column-3", "a", "b"},
		},
		{
			name: "double-dash terminator stops translation",
			args: []string{"comm", "--", "-1", "-2"},
			want: []string{"comm", "--", "-1", "-2"},
		},
		{
			name: "non-column tokens pass through unchanged",
			args: []string{"comm", "--version", "-", "-1x", "file"},
			want: []string{"comm", "--version", "-", "-1x", "file"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateColumnFlags(tc.args)
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("translateColumnFlags(%q) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
