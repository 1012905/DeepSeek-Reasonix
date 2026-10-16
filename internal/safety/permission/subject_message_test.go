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

func TestTheMessageSubjectIsAGenericRuleNotSpecificToOneTool(t *testing.T) {
	both, _ := json.Marshal(map[string]string{"to": "alice@example.test", "text": "hello"})
	if got, want := Subject(both), "alice@example.test: hello"; got != want {
		t.Fatalf("any tool call carrying string to and text gets that subject: %q, want %q", got, want)
	}
	for name, args := range map[string]string{
		"only text":       `{"text":"hello"}`,
		"non-string to":   `{"to":5,"text":"hello"}`,
		"empty to":        `{"to":"","text":"hello"}`,
		"other arguments": `{"recipient":"x","body":"y"}`,
		"a command wins":  `{"command":"ls","to":"a","text":"b"}`,
	} {
		got := Subject(json.RawMessage(args))
		want := ""
		if name == "a command wins" {
			want = "ls"
		}
		if got != want {
			t.Errorf("%s: Subject = %q, want %q", name, got, want)
		}
	}
}
