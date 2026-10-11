package textutil

import (
	"strings"
	"testing"
)

const hostile = "run​ ‮evil\x1b]0;title\x07 x⁠y"

// Each policy answers one question the same way wherever it is applied: the
// locator policy shows every hidden character, the prose policy drops them.
func TestPoliciesAgreeOnOneHostileFixture(t *testing.T) {
	shown := ShownLocator(hostile)
	for _, hid := range []string{"​", "‮", "\x1b", "\x07", "⁠"} {
		if strings.Contains(shown, hid) {
			t.Fatalf("locator policy left %q visible-as-invisible: %q", hid, shown)
		}
	}
	for _, want := range []string{`\u{200b}`, `\u{202e}`, `\u{1b}`, `\u{2060}`} {
		if !strings.Contains(shown, want) {
			t.Fatalf("locator policy hid %s: %q", want, shown)
		}
	}
	prose := strings.Join(strings.Fields(ShownProse(hostile)), " ")
	display := SanitizeDisplay(hostile)
	if display != prose {
		t.Fatalf("the display surface reads %q but the prose surface reads %q", display, prose)
	}
	if launch := SanitizeLaunch(hostile); strings.ContainsAny(launch, "​‮⁠\x1b") {
		t.Fatalf("launch policy kept an invisible character: %q", launch)
	}
	if !Obscures(hostile) || Obscures("plain text 日本語 👨‍👩‍👧") {
		t.Fatal("Obscures must flag hostile text and only that")
	}
}

func TestSanitizeDisplayDropsBidiControlsLikeTheProsePolicy(t *testing.T) {
	if got := SanitizeDisplay("ab‮cd⁦ef"); got != "abcdef" {
		t.Fatalf("got %q", got)
	}
}
