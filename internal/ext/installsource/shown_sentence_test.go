package installsource

import (
	"strings"
	"testing"
)

func TestSkippedMarketplaceWarningShowsAnUnterminatedEscapeInsteadOfEatingIt(t *testing.T) {
	got := skippedMarketplaceWarning("demo", "bad source\x1b]0;")
	if strings.ContainsRune(got, 0x1b) || !strings.Contains(got, `\u{1b}]0;`) || !strings.HasPrefix(got, `skipped Claude marketplace plugin "demo": `) {
		t.Fatalf("got %q", got)
	}
}
