package boot

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/contract/event"
	"reasonix/internal/contract/provider"
)

type jobListProvider struct {
	mu   sync.Mutex
	reqs []provider.Request
}

func (p *jobListProvider) Name() string { return "boot-effect-joblist" }

func (p *jobListProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.mu.Lock()
	p.reqs = append(p.reqs, req)
	round := len(p.reqs)
	p.mu.Unlock()
	ch := make(chan provider.Chunk, 3)
	switch round {
	case 1:
		ch <- provider.Chunk{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Arguments: `{"command":"sleep 30","run_in_background":true}`}}
	case 2:
		ch <- provider.Chunk{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c2", Name: "bash_output", Arguments: `{}`}}
	default:
		ch <- provider.Chunk{Type: provider.ChunkText, Text: "ok"}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

// TestEffectBashOutputListsRunningJobsThroughRealBuild pins what the model is
// told about its own background work: the host's enumeration of running jobs
// reaches the next provider request, instead of a count kept in the model's head.
func TestEffectBashOutputListsRunningJobsThroughRealBuild(t *testing.T) {
	isolateConfigHome(t)
	dir := robustTempDir(t)
	t.Chdir(dir)
	rec := &jobListProvider{}
	provider.Register("boot-effect-joblist", func(provider.Config) (provider.Provider, error) { return rec, nil })
	writeUserConfig(t, "[sandbox]\nbash = \"off\"\n")
	writeFile(t, dir, "reasonix.toml", `
default_model = "test-model"

[agent]
system_prompt = "BASE"

[[providers]]
name = "test-model"
kind = "boot-effect-joblist"
model = "x"
`)
	approveWorkspace(t, dir)
	ctrl, err := Build(context.Background(), Options{Sink: event.Discard})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer ctrl.Close()
	ctrl.SetSessionPath(filepath.Join(dir, ".reasonix", "sessions", "joblist.jsonl"))
	ctrl.SetBypass(true)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := ctrl.Run(ctx, "start a job and list jobs"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.reqs) < 3 {
		t.Fatalf("provider saw %d rounds, want 3", len(rec.reqs))
	}
	var result string
	for _, m := range rec.reqs[2].Messages {
		if m.Role == provider.RoleTool && m.ToolCallID == "c2" {
			result = m.Content
		}
	}
	if !strings.Contains(result, "Running background jobs (1)") || !strings.Contains(result, "sleep 30") {
		t.Fatalf("bash_output {} result = %q", result)
	}
	for _, tl := range rec.reqs[0].Tools {
		if tl.Name == "bash_output" && strings.Contains(string(tl.Parameters), `"job_id"]`) {
			t.Fatalf("bash_output still requires job_id: %s", tl.Parameters)
		}
	}
}
