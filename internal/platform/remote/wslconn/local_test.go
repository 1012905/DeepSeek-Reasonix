//go:build linux

package wslconn

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"testing"

	"reasonix/internal/platform/wsl"
)

// localCommand answers wsl.exe's --exec by running the program here, so the
// scripts meet a real GNU userland without a distribution.
func localCommand(t *testing.T) wsl.Command {
	t.Helper()
	return func(ctx context.Context, stdin io.Reader, args ...string) (wsl.Result, error) {
		for i, a := range args {
			if a == "--exec" {
				args = args[i+1:]
				break
			}
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		var out, errb bytes.Buffer
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, &out, &errb
		err := cmd.Run()
		res := wsl.Result{Stdout: out.Bytes(), Stderr: errb.Bytes()}
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode, err = ee.ExitCode(), nil
		}
		return res, err
	}
}

func TestFileLayerAgainstALocalPOSIXShell(t *testing.T) {
	if err := exec.Command("stat", "-c", "%s", "/").Run(); err != nil {
		t.Skip("needs GNU stat")
	}
	c := &Conn{distro: "local", run: localCommand(t)}
	runFileScenario(t, c, t.TempDir())
}
