//go:build windows

package wslconn

import (
	"context"
	"os"
	"testing"

	"reasonix/internal/platform/wsl"
)

// REASONIX_LIVE_WSL names an installed WSL2 distribution to run the file layer in.
func TestFileLayerInARealDistribution(t *testing.T) {
	name := os.Getenv("REASONIX_LIVE_WSL")
	if name == "" {
		t.Skip("set REASONIX_LIVE_WSL to an installed WSL2 distribution")
	}
	ctx := context.Background()
	c, err := New(ctx, wsl.System(), wsl.SystemCommand(), name)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Exec(ctx, "mktemp -d")
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("mktemp = %+v, %v", res, err)
	}
	dir := string(res.Stdout[:len(res.Stdout)-1])
	defer c.Exec(ctx, "rm -rf '"+dir+"'")
	runFileScenario(t, c, dir)
}
