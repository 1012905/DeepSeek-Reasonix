package skill

import (
	"testing"

	"reasonix/internal/base/testenv"
)

func TestSpecDescriptiveFieldsAreReadAsWritten(t *testing.T) {
	home := testenv.TempDir(t)
	writeSkill(t, home, ".reasonix/skills/pdf-processing/SKILL.md", "---\nname: pdf-processing\ndescription: pdfs\nlicense: Apache-2.0\ncompatibility: Requires git and network access\nmetadata:\n  author: example-org\n  version: \"1.0\"\n  nested:\n    skipped: yes\n---\nBODY")
	sk, ok := New(Options{HomeDir: home, DisableBuiltins: true}).Read("pdf-processing")
	if !ok {
		t.Fatal("skill did not load")
	}
	if sk.License != "Apache-2.0" || sk.Compatibility != "Requires git and network access" {
		t.Fatalf("license/compatibility = %q / %q", sk.License, sk.Compatibility)
	}
	if len(sk.Metadata) != 2 || sk.Metadata["author"] != "example-org" || sk.Metadata["version"] != "1.0" {
		t.Fatalf("metadata = %v, want the two string values and not the nested map", sk.Metadata)
	}
}

func TestASkillWithoutTheDescriptiveFieldsCarriesNone(t *testing.T) {
	home := testenv.TempDir(t)
	writeSkill(t, home, ".reasonix/skills/plain.md", "---\nname: plain\ndescription: d\n---\nBODY")
	sk, _ := New(Options{HomeDir: home, DisableBuiltins: true}).Read("plain")
	if sk.License != "" || sk.Compatibility != "" || sk.Metadata != nil {
		t.Fatalf("unexpected fields: %+v", sk)
	}
}
