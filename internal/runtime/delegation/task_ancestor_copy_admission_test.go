package delegation

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"reasonix/internal/contract/provider"
	"reasonix/internal/contract/tool"
	"reasonix/internal/runtime/agent"
	"reasonix/internal/state/sessionstore"
)

func TestTaskAncestorCopyAdmissionBeforeRunningMetadata(t *testing.T) {
	for _, route := range []string{"continue_from", "fork_from"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			store := NewSubagentStore(filepath.Join(dir, "subagents"))
			prov := &mockProvider{name: "sub", chunks: []provider.Chunk{
				{Type: provider.ChunkText, Text: "persisted source answer"},
				{Type: provider.ChunkDone},
			}}
			task := NewTaskTool(prov, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, "", "sys", nil, 0, "", "", nil).
				WithTranscripts(store, t.TempDir(), "base-model", "base-effort")
			rootCtx := agent.WithParentSession(context.Background(), "root")
			out, err := task.Execute(rootCtx, []byte(`{"prompt":"source task"}`))
			if err != nil {
				t.Fatal(err)
			}
			source := subagentRefFromOutput(t, out)
			saveTestBranchMeta(t, dir, "root", "")
			saveTestBranchMeta(t, dir, "child", "root")

			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			var probes atomic.Int32
			// This existing host callback is consulted by MarkRunning before it
			// publishes metadata, after the real TaskTool prepared its transcript.
			store.WithDestroyedChecker(func(parent string) bool {
				if parent == "child" && probes.Add(1) == 1 {
					close(entered)
					<-release
				}
				return false
			})
			childCtx := agent.WithParentSession(context.Background(), "child")
			type result struct {
				output string
				err    error
			}
			done := make(chan result, 1)
			go func() {
				output, err := task.Execute(childCtx, []byte(`{"prompt":"first child","continue_from":"`+source+`"}`))
				done <- result{output, err}
			}()
			defer unblock()
			select {
			case <-entered:
			case <-time.After(10 * time.Second):
				t.Fatal("first TaskTool call did not reach metadata publication")
			}
			_, secondErr := task.Execute(childCtx, []byte(`{"prompt":"competing child","`+route+`":"`+source+`"}`))
			unblock()
			first := <-done
			if first.err != nil {
				t.Fatalf("first TaskTool: %v", first.err)
			}
			if secondErr == nil || !strings.Contains(secondErr.Error(), "already") {
				t.Errorf("second TaskTool admission = %v; want busy refusal", secondErr)
			}
			copies, err := sessionstore.ListSubagentsByParent(dir, "child")
			if err != nil {
				t.Fatal(err)
			}
			if len(copies) != 1 || copies[0].Ref != subagentRefFromOutput(t, first.output) {
				t.Errorf("current-parent copies = %d; want only the first admitted run", len(copies))
			}
			if len(prov.requests) != 2 {
				t.Errorf("provider requests = %d; want source and first child only", len(prov.requests))
			}
			found := false
			for _, message := range prov.lastReq.Messages {
				found = found || message.Content == "persisted source answer"
			}
			if !found {
				t.Fatal("child did not receive the persisted source transcript")
			}
		})
	}
}
