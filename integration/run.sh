#!/bin/sh
# Integration checks for yup-comm, run inside a Debian (GNU coreutils) container.
#
# comm compares two SORTED files line by line and emits three columns: lines
# only in FILE1, lines only in FILE2 (one tab), and lines common to both (two
# tabs). The -1/-2/-3 flags suppress a column and collapse its leading tab.
#
# parity ARGS...  — yup-comm must produce byte-identical output to GNU `comm`
#                   for the same two sorted operand files plus flags.
set -eu

fails=0

# Two sorted inputs with unique-to-1, unique-to-2, and common lines.
printf 'apple\nbanana\ncherry\n' >/tmp/a.txt
printf 'banana\ncherry\nkiwi\n' >/tmp/b.txt

parity() {
  ours=$(yup-comm "$@" /tmp/a.txt /tmp/b.txt 2>/dev/null || true)
  gnu=$(comm "$@" /tmp/a.txt /tmp/b.txt 2>/dev/null || true)
  if [ "$ours" = "$gnu" ]; then
    printf 'ok    parity  comm %s\n' "$*"
  else
    printf 'FAIL  parity  comm %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
    fails=$((fails + 1))
  fi
}

# Default three-column output.
parity
# Single-column suppressions.
parity -1
parity -2
parity -3
# Paired suppressions, separate flags.
parity -1 -2
parity -2 -3
# All three suppressed (empty output).
parity -1 -2 -3
# Grouped digit forms (GNU comm accepts -12, -23, -123).
parity -12
parity -23
parity -123

if [ "$fails" -ne 0 ]; then
  printf '\n%s check(s) failed\n' "$fails"
  exit 1
fi
printf '\nall checks passed\n'
