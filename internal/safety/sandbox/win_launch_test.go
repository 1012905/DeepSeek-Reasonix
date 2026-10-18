package sandbox

import (
	"slices"
	"strings"
	"testing"
)

func TestALauncherRequestRoundTripsAndOnlyThatRequestIsOne(t *testing.T) {
	sid := newSyntheticSID()
	argv := winLaunchArgv(`C:\Program Files\Reasonix\reasonix.exe`, winLaunch{SID: sid, Env: map[string]string{"TEMP": `D:\t`}}, []string{"bash", "-c", "echo hi"})
	l, rest, ok := parseWinLaunch(argv)
	if !ok || l.SID != sid || l.Env["TEMP"] != `D:\t` || !slices.Equal(rest, []string{"bash", "-c", "echo hi"}) {
		t.Fatalf("parse = %+v %v %v", l, rest, ok)
	}
	for name, args := range map[string][]string{
		"ordinary start":    {"reasonix", "chat"},
		"no command":        {"reasonix", winLaunchFlag, "e30", "--"},
		"missing separator": {"reasonix", winLaunchFlag, "e30", "x", "bash"},
		"not base64":        {"reasonix", winLaunchFlag, "***", "--", "bash"},
		"flag not first":    {"reasonix", "x", winLaunchFlag, "e30", "--", "bash"},
	} {
		if _, _, ok := parseWinLaunch(args); ok {
			t.Errorf("%s parsed as a launcher request", name)
		}
	}
}

// A request naming a SID this package could not have made is not honoured: it
// would restrict a token to a real account, and that is no restriction.
func TestALauncherRequestNamingARealAccountIsRefused(t *testing.T) {
	for _, sid := range []string{"S-1-1-0", "S-1-5-18", "S-1-5-21-1-2-3-1001", ""} {
		argv := winLaunchArgv("reasonix", winLaunch{SID: sid}, []string{"cmd"})
		if _, _, ok := parseWinLaunch(argv); ok {
			t.Errorf("a request for %q was accepted", sid)
		}
	}
}

func TestEnvOverridesReplaceByCaseInsensitiveNameAndComeOutInOrder(t *testing.T) {
	got := mergeEnv([]string{"Path=C:\\bin", "temp=C:\\old", "KEEP=1"}, map[string]string{"TEMP": `D:\new`, "TMP": `D:\new`})
	want := []string{"Path=C:\\bin", "KEEP=1", `TEMP=D:\new`, `TMP=D:\new`}
	if !slices.Equal(got, want) {
		t.Fatalf("env = %v, want %v", got, want)
	}
	if strings.Count(strings.ToLower(strings.Join(got, "\n")), "temp=") != 1 {
		t.Fatalf("TEMP appears more than once: %v", got)
	}
}
