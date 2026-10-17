package agentcomposition

import (
	"errors"
	"testing"
)

func TestDefinitionsValidateIdentityAndSelections(t *testing.T) {
	d := Builtins()[0]
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	d.ID = "user:../escape"
	if !errors.Is(d.Validate(), ErrInvalid) {
		t.Fatal("path identity accepted")
	}
	d = Builtins()[1]
	d.Skills = Selection{Mode: Select, Members: []Module{{ID: "one"}, {ID: "one"}}}
	if !errors.Is(d.Validate(), ErrInvalid) {
		t.Fatal("duplicate member accepted")
	}
}

func TestRevisionCanonicalizesParameterObjects(t *testing.T) {
	a := Builtins()[0]
	a.MCP = Selection{Mode: Select, Members: []Module{{ID: "example", Parameters: []byte(`{"b":2,"a":1}`)}}}
	b := a
	b.MCP.Members = []Module{{ID: "example", Parameters: []byte(`{"a":1,"b":2}`)}}
	if Revision(a) != Revision(b) {
		t.Fatal("object order changed revision")
	}
	b.Name = "Changed"
	if Revision(a) == Revision(b) {
		t.Fatal("definition edit did not change revision")
	}
}

func TestRestrictedTemplatesHaveNoAutomationOrDelegation(t *testing.T) {
	for _, d := range Builtins()[1:] {
		for _, s := range []Selection{d.Skills, d.MCP, d.Extensions, d.Hooks, d.Commands} {
			if s.Mode != Select || len(s.Members) != 0 {
				t.Fatalf("%s inherits modules", d.ID)
			}
		}
		for _, m := range d.Tools.Members {
			if m.ID == "bash" || m.ID == "task" || m.ID == "write_file" {
				t.Fatalf("%s carries %s", d.ID, m.ID)
			}
		}
	}
}
