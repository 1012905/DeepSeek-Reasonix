package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

// writeKilledAgentStub writes a stand-in for the reasonix binary that leaves
// exactly what a killed child leaves behind: no final metrics record, but the
// in-flight snapshot the metrics sink writes every two seconds.
func writeKilledAgentStub(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "stub-agent.sh")
	script := "#!/bin/sh\n" +
		"metrics=\"\"\n" +
		"prev=\"\"\n" +
		"for arg in \"$@\"; do\n" +
		"  if [ \"$prev\" = \"--metrics\" ]; then metrics=\"$arg\"; fi\n" +
		"  prev=\"$arg\"\n" +
		"done\n" +
		"printf '%s' '{\"prompt_tokens\":1200,\"completion_tokens\":80,\"steps\":9," +
		"\"cost\":0.0123,\"currency\":\"USD\",\"complete\":false}' > \"$metrics.partial\"\n" +
		"exit 137\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

func killedRun(t *testing.T) result {
	t.Helper()
	requireShellStub(t)
	work := testenv.TempDir(t)
	tk := task{ID: "stub-task", Prompt: "do the thing"}
	r := result{task: tk}
	cfg := suiteConfig{bin: writeKilledAgentStub(t, work)}
	if err := runSegments(context.Background(), cfg, tk, work, "", nil, &r); err == nil {
		t.Fatal("the stub exits 137; a leg that died must be reported as failed")
	}
	return r
}

// A run whose final metrics never landed has no harness-side accounting. The
// report already has the state for that (Unaccounted) and swebench mode sets it;
// the suite path must not fold such a run in as an accounted zero, or every
// published total silently omits what it spent.
func TestSuiteRunWithoutFinalMetricsIsNotCountedAsAccounted(t *testing.T) {
	r := killedRun(t)
	stats := gatherSuiteStats([]result{r})
	if stats.unaccounted != 1 {
		t.Fatalf("a run that wrote no final metrics counted as accounted:\n"+
			"  unaccounted=%d accounted=%d tokens=%d cost=%.4f\n"+
			"  (prompt_tokens=%d completion_tokens=%d, from the result itself)",
			stats.unaccounted, stats.accounted, stats.pTok+stats.cTok, stats.cost,
			r.PromptTokens, r.CompletionTokens)
	}
}

// The killed child did leave its numbers: the metrics sink snapshots in flight
// into <metrics>.partial. Swebench mode reads that sidecar; the suite path
// deletes the directory without looking, so the evidence is destroyed rather
// than being missing.
func TestSuiteRunRecoversTheInFlightSnapshotItsChildLeft(t *testing.T) {
	r := killedRun(t)
	if r.PromptTokens != 1200 || r.CompletionTokens != 80 {
		t.Fatalf("the child's in-flight snapshot was dropped:\n"+
			"  prompt_tokens=%d completion_tokens=%d cost=%.4f partial=%v unaccounted=%v\n"+
			"  want the 1200/80 the <metrics>.partial sidecar carries",
			r.PromptTokens, r.CompletionTokens, r.Cost, r.Partial, r.Unaccounted)
	}
}

// The meter-vs-harness divergence line divides one side by the other. A run
// whose harness-side number never landed has no side, so folding its meter
// tokens against a silent zero invents a divergence and blames the harness for
// it.
func TestMeterDivergenceDoesNotPriceAMetricslessRun(t *testing.T) {
	r := result{task: task{ID: "killed"}, Meter: &meterUsage{PromptTokens: 1200, CompletionTokens: 80}}
	got := renderMeterAccounting([]result{r})
	if strings.Contains(got, "self-report divergence") {
		t.Fatalf("a run with no harness-side number was priced as a self-report divergence:\n%s", got)
	}
}
