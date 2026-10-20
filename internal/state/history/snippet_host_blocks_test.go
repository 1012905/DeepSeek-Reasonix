package history

import (
	"testing"

	"reasonix/internal/contract/provider"
	"reasonix/internal/state/historycatalog"
	"reasonix/internal/state/sessionstore"
)

// The other half of the drift pin: both snippet builders must read a user turn
// exactly as the indexer does. If either reverts to raw Content, this fails
// while historycatalog's pin keeps the index honest.
func TestSnippetBuildersReadTheIndexedUserText(t *testing.T) {
	raw := "<response-language>\nFinal answer language preference: use English.\n</response-language>\n\n" +
		"zebra question about the migration order"
	msgs := []provider.Message{{Role: provider.RoleUser, Content: raw, RawContent: raw}}
	want := sessionstore.UserSaidText(raw)

	got, ok := candidateText(msgs, historycatalog.Candidate{MessageIndex: 0, Kind: string(KindUserText)})
	if !ok || got != want {
		t.Fatalf("candidateText = %q ok=%v, want %q", got, ok, want)
	}

	docs := extractDocuments(sourceFile{path: "blocked.jsonl"}, msgs, map[Kind]bool{KindUserText: true}, "")
	if len(docs) != 1 || docs[0].text != want {
		t.Fatalf("extractDocuments = %#v, want one document whose text is %q", docs, want)
	}
}
