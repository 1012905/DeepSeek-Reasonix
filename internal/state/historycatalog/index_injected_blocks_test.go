package historycatalog

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/retrieval"
	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/provider"
	"reasonix/internal/state/sessionstore"
)

// The transcript keeps the host's injected blocks in Content (that is how the
// provider receives them), and every display/search path strips them. The FTS
// projection must index the same text (sessionstore.UserSaidText), or a query
// for block-only words matches sessions whose snippet cannot contain them.
func TestFTSIndexesWhatTheUserSaidNotTheHostBlocks(t *testing.T) {
	ctx := context.Background()
	root := testenv.TempDir(t)
	path := filepath.Join(root, "blocked.jsonl")
	// The shape the runtime persists: <response-language> leads the turn, the
	// person's own words follow it.
	raw := "<response-language>\nFinal answer language preference: use English.\n</response-language>\n\n" +
		"zebra question about the migration order"
	saveMessages(t, path, provider.Message{Role: provider.RoleUser, Content: raw, RawContent: raw})

	catalog, err := Open(ctx, Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.Close(context.Background()) })
	if err := catalog.ReconcileRoot(ctx, Root{Path: root, Source: "project", Scope: "project", WorkspaceRoot: root}); err != nil {
		t.Fatal(err)
	}

	// Control: the person's own words must stay searchable.
	own, err := catalog.Search(ctx, SearchRequest{Query: "zebra migration", Roots: []string{root}})
	if err != nil || len(own.Items) != 1 {
		t.Fatalf("the user's own words are not searchable: items=%#v err=%v", own.Items, err)
	}
	// The block is host text. Flattening the transcript for display removes it,
	// so a search hit for these words shows a snippet that cannot contain them.
	stripped := sessionstore.DropLeadingTransientBlocks(raw)
	if strings.Contains(strings.ToLower(stripped), "language") || strings.Contains(strings.ToLower(stripped), "english") {
		t.Fatalf("fixture is wrong: the display text still contains the block: %q", stripped)
	}
	for _, query := range []string{"language preference", "english"} {
		got, err := catalog.Search(ctx, SearchRequest{Query: query, Roots: []string{root}})
		if err != nil {
			t.Fatalf("query %q: %v", query, err)
		}
		if len(got.Items) != 0 {
			t.Fatalf("query %q matched %d row(s) from a host-injected block that the displayed text strips; "+
				"the hit's snippet cannot contain %q (index and snippet disagree about what the turn says)",
				query, len(got.Items), query)
		}
	}
}

// The drift pin: one turn carries host blocks, and the terms the indexer stores
// must equal the terms of the text the snippet is built from. Editing either
// side to skip the shared strip makes these two strings disagree.
func TestIndexAndSnippetAgreeOnTheUserText(t *testing.T) {
	raw := "<response-language>\nFinal answer language preference: use English.\n</response-language>\n\n" +
		"zebra question about the migration order"
	docs := documents([]provider.Message{{Role: provider.RoleUser, Content: raw, RawContent: raw}})
	if len(docs) != 1 {
		t.Fatalf("documents() = %#v, want one user document", docs)
	}
	// The snippet is built from exactly this text (history/search.go,
	// history/indexed.go), so the two derived strings must be identical.
	snippet := strings.Join(retrieval.Tokens(sessionstore.UserSaidText(raw)), " ")
	if docs[0].terms != snippet {
		t.Fatalf("indexed terms = %q, snippet text terms = %q — the index and the snippet read the turn differently", docs[0].terms, snippet)
	}
}

// Control: a turn with no host blocks keeps its indexed terms byte for byte.
func TestIndexKeepsPlainUserTextUnchanged(t *testing.T) {
	const content = "plain zebra question about the migration order"
	docs := documents([]provider.Message{{Role: provider.RoleUser, Content: content}})
	if len(docs) != 1 {
		t.Fatalf("documents() = %#v, want one user document", docs)
	}
	want := strings.Join(retrieval.Tokens(content), " ")
	if docs[0].terms != want {
		t.Fatalf("plain turn indexed as %q, want %q", docs[0].terms, want)
	}
}
