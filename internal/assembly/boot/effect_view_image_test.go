package boot

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/event"
	"reasonix/internal/contract/provider"
)

// view_image is offered to a model that reads images and to no other: a session
// on a text-only model keeps exactly the tool block it had, and so its cached
// prompt prefix. What reaches the provider is checked, not the registry.
func TestEffectViewImageReachesOnlyAModelThatReadsImages(t *testing.T) {
	t.Parallel()
	for _, vision := range []bool{true, false} {
		t.Run(fmt.Sprintf("vision=%v", vision), func(t *testing.T) {
			t.Parallel()
			home := statedBootHome(t)
			dir := testenv.TempDir(t)
			rec := &peerToolProvider{}
			kind := fmt.Sprintf("boot-view-image-%s", strings.ReplaceAll(t.Name(), "/", "-"))
			provider.Register(kind, func(provider.Config) (provider.Provider, error) { return rec, nil })
			writeFile(t, dir, "reasonix.toml", fmt.Sprintf(`
default_model = "test-model"

[codegraph]
enabled = false

[[providers]]
name = "test-model"
kind = %q
model = "x"
vision = %v
`, kind, vision))
			approveWorkspaceAt(t, home, dir)
			ctrl, err := Build(context.Background(), Options{Home: home, WorkspaceRoot: dir, Sink: event.Discard})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(ctrl.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_ = ctrl.Run(ctx, "hi")
			has := false
			for _, name := range rec.toolNames() {
				has = has || name == "view_image"
			}
			if has != vision {
				t.Fatalf("vision=%v: tools in the first request = %v", vision, rec.toolNames())
			}
		})
	}
}
