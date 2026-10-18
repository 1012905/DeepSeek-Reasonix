package sandbox

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
)

// winLaunchFlag marks an argv as a request to run the rest of it under the
// write-restricted token. It is an argument of whichever host binary imports
// this package, so no helper has to be shipped and found.
const winLaunchFlag = "--reasonix-win-sandbox"

// winLaunch is what the launcher needs beyond the command: the SID to restrict
// the token to and the environment the host overrides. It carries no secret.
type winLaunch struct {
	SID string            `json:"sid"`
	Env map[string]string `json:"env,omitempty"`
}

// winLaunchArgv wraps argv so that running it starts self as the launcher.
func winLaunchArgv(self string, l winLaunch, argv []string) []string {
	raw, _ := json.Marshal(l)
	out := []string{self, winLaunchFlag, base64.RawURLEncoding.EncodeToString(raw), "--"}
	return append(out, argv...)
}

// parseWinLaunch reads a launcher request from os.Args-shaped input (program
// name first). ok is false for anything that is not one, so an ordinary start of
// the binary is untouched.
func parseWinLaunch(args []string) (l winLaunch, argv []string, ok bool) {
	if len(args) < 5 || args[1] != winLaunchFlag || args[3] != "--" {
		return winLaunch{}, nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(args[2])
	if err != nil || json.Unmarshal(raw, &l) != nil || !syntheticSID(l.SID) {
		return winLaunch{}, nil, false
	}
	return l, args[4:], true
}

// mergeEnv applies overrides over base. Windows names ignore case, so an
// override replaces the entry it would shadow wherever its spelling differs.
func mergeEnv(base []string, over map[string]string) []string {
	out := make([]string, 0, len(base)+len(over))
	set := map[string]bool{}
	for k := range over {
		set[strings.ToLower(k)] = true
	}
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		if !set[strings.ToLower(name)] {
			out = append(out, kv)
		}
	}
	keys := make([]string, 0, len(over))
	for k := range over {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out = append(out, k+"="+over[k])
	}
	return out
}
