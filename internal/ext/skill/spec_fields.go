package skill

import (
	"strings"

	"reasonix/internal/base/frontmatter"
)

// SpecFields are the specification's optional descriptive fields, read as
// written: where a skill comes from and what it expects.
type SpecFields struct {
	License       string
	Compatibility string
	Metadata      map[string]string
}

// applySpecFields reads the specification's descriptive fields: license and
// compatibility as text, metadata as a string-to-string map (a value that is not
// a plain string is left out, as the specification asks for string values).
func applySpecFields(sk *Skill, doc frontmatter.Document) {
	if v, ok := doc.Lookup("license"); ok && v.Kind == frontmatter.KindScalar {
		sk.License = strings.TrimSpace(v.Scalar)
	}
	if v, ok := doc.Lookup("compatibility"); ok && v.Kind == frontmatter.KindScalar {
		sk.Compatibility = strings.TrimSpace(v.Scalar)
	}
	v, ok := doc.Lookup("metadata")
	if !ok || v.Kind != frontmatter.KindMapping {
		return
	}
	for _, f := range v.Fields {
		if f.Value.Kind == frontmatter.KindScalar && strings.TrimSpace(f.Key) != "" {
			if sk.Metadata == nil {
				sk.Metadata = map[string]string{}
			}
			sk.Metadata[f.Key] = f.Value.Scalar
		}
	}
}
