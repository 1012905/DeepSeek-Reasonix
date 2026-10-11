// Display text that came from somewhere we do not control — an MCP server's
// own description of itself, a tool's docstring — reaches a terminal and a
// settings pane. Both need the same thing done to it first, and doing it in
// each frontend is how the two drift.
package textutil

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// SanitizeDisplay is the prose policy for one line: escape sequences and every
// character BoundProse removes are dropped (controls, bidirectional overrides,
// zero-width and format characters), and whitespace collapses to single spaces,
// so one string cannot repaint a terminal, reorder a row or smuggle a line break.
func SanitizeDisplay(s string) string {
	rs := []rune(strings.ToValidUTF8(ansi.Strip(s), "\uFFFD"))
	hidden := hiddenMask(rs)
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range rs {
		switch {
		case r == '\t' || r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029' || r == '\u0085':
			b.WriteByte(' ')
		case hidden[i]:
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// Obscures reports whether s holds anything the literal policy would show as
// \u{hex}: a surface that deletes such characters says so with it.
func Obscures(s string) bool {
	rs := []rune(strings.ToValidUTF8(s, "\uFFFD"))
	return slices.Contains(hiddenMask(rs), true)
}

// SanitizeLaunch is the launch-line policy: the display policy applied before
// a secret is matched, because a key split by a zero-width character would
// otherwise survive redaction. It deletes, so the caller that shows the result
// must say so when Obscures held for the input.
func SanitizeLaunch(s string) string {
	s = SanitizeDisplay(s)
	return strings.Map(func(r rune) rune {
		if isFormat(r) {
			return -1
		}
		return r
	}, s)
}
