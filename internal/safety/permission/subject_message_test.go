package permission

import (
	"encoding/json"
	"testing"
)

func TestAMessageToAnotherSessionIsApprovedByItsRecipientAndFullText(t *testing.T) {
	long := "line one\n" + string(make([]byte, 0)) + "the rest of the words that must not be cut"
	args, _ := json.Marshal(map[string]string{"to": "docs/notes", "text": long})
	if got, want := Subject(args), "docs/notes: "+long; got != want {
		t.Fatalf("Subject = %q, want %q", got, want)
	}
	only, _ := json.Marshal(map[string]string{"to": "docs/notes"})
	if got := Subject(only); got != "" {
		t.Fatalf("a call with no text is not a message: %q", got)
	}
}
