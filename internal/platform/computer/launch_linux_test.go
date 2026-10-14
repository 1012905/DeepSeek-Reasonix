package computer

import (
	"os"
	"testing"
)

func TestCUALinuxLaunchRejectsArgumentsAndCommandExpansion(t *testing.T) {
	for _, command := range []string{"/usr/bin/app --flag", "/usr/bin/app; /bin/sh", "/usr/bin/app$(touch /tmp/marker)", "'/tmp/app;touch${IFS}/tmp/marker'", "FOO=x /usr/bin/app", "/usr/bin/app > /tmp/out", "app", ""} {
		if _, err := nativeLaunchSpecification(nativeApplication{Bundle: "example.desktop", LaunchPath: command}); CodeOf(err) != CodeUnsupported {
			t.Fatalf("launch command accepted %q: %v", command, err)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := applicationLaunch(nativeApplication{Bundle: "example.desktop", LaunchPath: exe})
	if err != nil || spec.args["launch_path"] != spec.executable {
		t.Fatalf("native ELF launch: %+v %v", spec, err)
	}
}
