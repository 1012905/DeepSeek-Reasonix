package memory

import (
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

// The two cases the single-identity audit found outside its own scope: a name/id
// collision across scopes, and the legacy same-name other-scope archive path.
// They are reproduced here from the audit that found them.

// An unqualified update whose name is already active in more than one scope has
// no single identity to update: guessing would archive a fact the caller never
// named, so the store must refuse and say which facts collide.
func TestStoreV2UnqualifiedUpdateRefusesADuplicateNameAcrossScopes(t *testing.T) {
	root := testenv.TempDir(t)
	store := Store{Dir: filepath.Join(root, "project"), GlobalDir: filepath.Join(root, "global")}
	if _, err := store.SaveWithOptions(Memory{Name: "global/shared.md", Description: "global", Body: "global body"}, SaveOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveWithOptions(Memory{Name: "project/shared.md", Description: "project", Body: "project body"}, SaveOptions{}); err != nil {
		t.Fatal(err)
	}
	before, err := store.ListAll()
	if err != nil {
		t.Fatal(err)
	}

	_, saveErr := store.SaveWithOptions(Memory{Name: "shared", Description: "updated", Body: "project v2"}, SaveOptions{})
	if saveErr == nil {
		t.Fatal("an unqualified update of a name active in two scopes reported success")
	}
	if !strings.Contains(saveErr.Error(), "shared") {
		t.Fatalf("the refusal does not name the colliding name: %v", saveErr)
	}
	after, err := store.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("the refused update changed the store: %d active facts before, %d after", len(before), len(after))
	}
	for _, m := range after {
		if m.Body == "project v2" {
			t.Fatalf("the refused update was written anyway: %+v", m)
		}
	}
}

// A failed rename must not leave a revision that a later stale write can be
// accepted against by accident of the directory listing order.
func TestStoreV2FailedRenameLeavesNoAcceptableStaleRevision(t *testing.T) {
	for _, tc := range []struct{ oldName, newName string }{
		{"alpha", "beta"},
		{"zeta", "alpha"},
	} {
		t.Run(tc.oldName+"_to_"+tc.newName, func(t *testing.T) {
			store := Store{Dir: testenv.TempDir(t)}
			first, err := store.SaveWithOptions(Memory{Name: tc.oldName, Description: "first", Body: "v1"}, SaveOptions{})
			if err != nil {
				t.Fatal(err)
			}
			failRenames(t, func(path string) bool { return strings.HasSuffix(path, tc.oldName+".md") })

			if _, saveErr := store.SaveWithOptions(Memory{
				ID: first.Memory.ID, Name: tc.newName, Description: "renamed", Body: "v2",
			}, SaveOptions{ExpectedRevision: 1, RequireExpectedRevision: true}); saveErr == nil {
				t.Fatal("the rename reported success although the old name could not be retired")
			}
			stopFailing(store.Dir)

			// Whatever is on disk now, only one identity may claim this id, and a
			// revision-1 write must be judged against it rather than against a
			// file the directory listing happened to return first.
			var actives []Memory
			all, err := store.ListAll()
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range all {
				if m.ID == first.Memory.ID {
					actives = append(actives, m)
				}
			}
			if len(actives) > 1 {
				t.Fatalf("the failed rename left %d active identities for one id: %+v", len(actives), actives)
			}
			if len(actives) == 1 && actives[0].Revision != 1 {
				t.Fatalf("the failed rename left revision %d on disk: %+v", actives[0].Revision, actives[0])
			}
		})
	}
}
