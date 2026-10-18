package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

func TestRemoteHostKindDefaultsToSSH(t *testing.T) {
	for kind, want := range map[string]string{"": RemoteKindSSH, " SSH ": RemoteKindSSH, "wsl": RemoteKindWSL, "WSL": RemoteKindWSL, "docker": "docker"} {
		if got := (RemoteHostEntry{Kind: kind}).HostKind(); got != want {
			t.Errorf("HostKind(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestUpsertRemoteHostHoldsAWSLEntryToItsOwnFields(t *testing.T) {
	cfg := Default()
	bad := map[string]RemoteHostEntry{
		"no distro":        {Name: "w", Kind: "wsl"},
		"blank distro":     {Name: "w", Kind: "wsl", Distro: "  "},
		"newline":          {Name: "w", Kind: "wsl", Distro: "Ubuntu\nx"},
		"an address":       {Name: "w", Kind: "wsl", Distro: "Ubuntu", Host: "198.51.100.4"},
		"a port":           {Name: "w", Kind: "wsl", Distro: "Ubuntu", Port: 22},
		"a login":          {Name: "w", Kind: "wsl", Distro: "Ubuntu", User: "dev"},
		"a key":            {Name: "w", Kind: "wsl", Distro: "Ubuntu", IdentityFile: "~/.ssh/id"},
		"a passphrase":     {Name: "w", Kind: "wsl", Distro: "Ubuntu", PassphraseEnv: "X"},
		"a password":       {Name: "w", Kind: "wsl", Distro: "Ubuntu", PasswordEnv: "X"},
		"a jump":           {Name: "w", Kind: "wsl", Distro: "Ubuntu", ProxyJump: "j"},
		"ssh_config":       {Name: "w", Kind: "wsl", Distro: "Ubuntu", UseSSHConfig: true},
		"a forward":        {Name: "w", Kind: "wsl", Distro: "Ubuntu", Forwards: []RemoteForwardEntry{{Type: "local", Bind: "8080", Target: "h:80"}}},
		"bad install":      {Name: "w", Kind: "wsl", Distro: "Ubuntu", ServeInstall: "curlpipe"},
		"unknown kind":     {Name: "w", Kind: "docker", Host: "h"},
		"distro on an ssh": {Name: "w", Host: "h", Distro: "Ubuntu"},
	}
	for name, e := range bad {
		if err := cfg.UpsertRemoteHost(e); err == nil {
			t.Errorf("%s: accepted %+v", name, e)
		}
	}
	if len(cfg.Remote.Hosts) != 0 {
		t.Fatalf("invalid hosts persisted: %+v", cfg.Remote.Hosts)
	}
	if err := cfg.UpsertRemoteHost(RemoteHostEntry{Name: "w", Kind: "wsl", Distro: "Ubuntu-24.04", Workspace: "/home/dev/app", ServeInstall: "upload"}); err != nil {
		t.Fatalf("a valid wsl entry was refused: %v", err)
	}
}

func TestAWSLHostSurvivesSaveAndReload(t *testing.T) {
	isolateUserConfigHome(t)
	home := testenv.TempDir(t)
	t.Setenv("REASONIX_HOME", home)
	path := filepath.Join(home, "config.toml")
	if err := os.WriteFile(path, []byte("default_model = \"deepseek\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := LoadForEdit(path)
	if err := cfg.UpsertRemoteHost(RemoteHostEntry{Name: "dev", Kind: "wsl", Distro: "Ubuntu", Workspace: "/home/dev/app"}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.UpsertRemoteHost(RemoteHostEntry{Name: "box", Host: "198.51.100.4"}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if !strings.Contains(text, `kind = "wsl"`) || !strings.Contains(text, `distro = "Ubuntu"`) || strings.Contains(text, `host = ""`) {
		t.Fatalf("the saved wsl entry is wrong:\n%s", text)
	}
	if strings.Count(text, "kind =") != 1 {
		t.Fatalf("an ssh entry gained a kind line:\n%s", text)
	}
	got, ok := LoadForEdit(path).RemoteHost("dev")
	if !ok || got.HostKind() != RemoteKindWSL || got.Distro != "Ubuntu" || got.Workspace != "/home/dev/app" {
		t.Fatalf("reloaded = %+v, %v", got, ok)
	}
	if box, _ := LoadForEdit(path).RemoteHost("box"); box.HostKind() != RemoteKindSSH {
		t.Fatalf("the ssh entry changed kind: %+v", box)
	}
}

func TestRemoteTargetIsSpelledPerKind(t *testing.T) {
	for want, e := range map[string]RemoteHostEntry{
		"dev@box":       {Host: "box", User: "dev"},
		"box":           {Host: "box", Port: 22},
		"dev@box:2200":  {Host: "box", User: "dev", Port: 2200},
		"wsl:Ubuntu":    {Kind: "wsl", Distro: "Ubuntu"},
		"wsl:Debian-12": {Kind: "WSL", Distro: "Debian-12"},
	} {
		if got := e.Target(); got != want {
			t.Errorf("Target(%+v) = %q, want %q", e, got, want)
		}
	}
}
