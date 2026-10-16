package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

// stubScript builds a stand-in agent that records the --metrics path it was
// given, writes the metrics body it was built with to that path plus suffix,
// and exits with code. A non-empty suffix is what a killed child leaves: the
// in-flight snapshot and no final record.
func stubScript(t *testing.T, name, body, suffix string, code int) string {
	t.Helper()
	requireShellStub(t)
	script := fmt.Sprintf(`#!/bin/sh
metrics=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--metrics" ]; then metrics="$arg"; fi
  prev="$arg"
done
printf '%%s' "$metrics" > metrics-path.txt
printf '%%s' '%s' > "$metrics%s"
exit %d
`, body, suffix, code)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

// recordedMetricsPath reads back the --metrics path the stub was handed.
func recordedMetricsPath(t *testing.T, work string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(work, "metrics-path.txt"))
	if err != nil {
		t.Fatalf("the stub did not record its metrics path: %v", err)
	}
	got := strings.TrimSpace(string(b))
	if got == "" {
		t.Fatal("the stub recorded an empty metrics path")
	}
	return got
}

// A run that lands its final metrics record is untouched by the recovery
// path: it stays accounted, keeps its real numbers and is still not partial.
func TestSuiteHealthyRunKeepsItsFinalAccount(t *testing.T) {
	work := testenv.TempDir(t)
	tk := task{ID: "healthy-task", Prompt: "do the thing"}
	body := `{"prompt_tokens":900,"completion_tokens":60,"steps":4,"cost":0.02,` +
		`"currency":"USD","complete":true}`
	cfg := suiteConfig{bin: stubScript(t, "stub-ok.sh", body, "", 0)}
	r := result{task: tk}
	if err := runSegments(context.Background(), cfg, tk, work, "", nil, &r); err != nil {
		t.Fatalf("a healthy leg must not fail the run: %v", err)
	}
	stats := gatherSuiteStats([]result{r})
	if stats.unaccounted != 0 || stats.accounted != 1 {
		t.Fatalf("a healthy run changed category: unaccounted=%d accounted=%d",
			stats.unaccounted, stats.accounted)
	}
	if stats.pTok != 900 || stats.cTok != 60 || stats.cost != 0.02 {
		t.Fatalf("a healthy run's numbers changed: tokens=%d/%d cost=%.4f",
			stats.pTok, stats.cTok, stats.cost)
	}
	if r.Partial || r.Unaccounted {
		t.Fatalf("a healthy run was flagged: partial=%v unaccounted=%v", r.Partial, r.Unaccounted)
	}
	// The metrics directory is disposable once the final record was read.
	dir := filepath.Dir(recordedMetricsPath(t, work))
	if dirExists(dir) {
		t.Fatalf("a healthy run left its metrics directory behind: %s", dir)
	}
}

// The snapshot a killed child leaves is the only evidence of what it spent, so
// recovering the numbers must not also destroy the file they came from.
func TestSuiteRunKeepsTheSnapshotItRecovered(t *testing.T) {
	work := testenv.TempDir(t)
	tk := task{ID: "killed-task", Prompt: "do the thing"}
	body := `{"prompt_tokens":1200,"completion_tokens":80,"steps":9,"cost":0.0123,` +
		`"currency":"USD","complete":false}`
	cfg := suiteConfig{bin: stubScript(t, "stub-partial.sh", body, ".partial", 137)}
	r := result{task: tk}
	if err := runSegments(context.Background(), cfg, tk, work, "", nil, &r); err == nil {
		t.Fatal("the stub exits 137; a leg that died must be reported as failed")
	}
	path := recordedMetricsPath(t, work)
	if fileExists(path) {
		t.Fatalf("a killed child cannot have written a final record: %s", path)
	}
	if !fileExists(path + ".partial") {
		t.Fatalf("the recovered snapshot was deleted with the metrics directory: %s.partial", path)
	}
	if r.PromptTokens != 1200 || r.CompletionTokens != 80 {
		t.Fatalf("the snapshot was kept but not recovered: tokens=%d/%d",
			r.PromptTokens, r.CompletionTokens)
	}
}

// TestMeterAccountingSkipsAnUnaccountedRunWithTokens pins the other term of the
// guard: a run the suite already marked unmeasured stays out of the ratio even
// when the harness number is present, so a partial account is not priced.
func TestMeterAccountingSkipsAnUnaccountedRunWithTokens(t *testing.T) {
	r := result{
		task:        task{ID: "killed"},
		runMetrics:  runMetrics{PromptTokens: 100, CompletionTokens: 10},
		Unaccounted: true,
		Meter:       &meterUsage{PromptTokens: 1200, CompletionTokens: 80},
	}
	got := renderMeterAccounting([]result{r})
	if strings.Contains(got, "self-report divergence") {
		t.Fatalf("an unaccounted run with a harness number was priced as a self-report divergence:\n%s", got)
	}
}
