package computer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCUAWindowsPathIdentityUsesOSExecutableName(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bundle, actual, err := processIdentity(int32(os.Getpid()), exe)
	if err != nil || bundle != filepath.Base(exe) || actual != filepath.Base(exe) {
		t.Fatalf("bundle=%q actual=%q err=%v", bundle, actual, err)
	}
	const aumid = "Example.Package!Application"
	bundle, _, err = processIdentity(int32(os.Getpid()), aumid)
	if err != nil || bundle != aumid {
		t.Fatalf("changed package identity: %q %v", bundle, err)
	}
}
