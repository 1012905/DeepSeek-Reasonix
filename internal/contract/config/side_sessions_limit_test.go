package config

import (
	"slices"
	"testing"
)

func TestSideSessionLimitIsUnlimitedUnlessTheUserSetsOne(t *testing.T) {
	for in, want := range map[int]int{-4: 0, 0: 0, 3: 3, 40: 40} {
		cfg := &Config{}
		cfg.Agent.MaxSideSessions = in
		if got := cfg.SideSessionLimit(); got != want {
			t.Errorf("max_side_sessions %d -> %d, want %d", in, got, want)
		}
	}
	if Default().SideSessionLimit() != 0 {
		t.Error("the default limits side conversations")
	}
	var none *Config
	if none.SideSessionLimit() != 0 {
		t.Error("a nil config limits side conversations")
	}
}

func TestProjectCannotSetTheSideSessionLimit(t *testing.T) {
	cfg, _ := loadScoped(t, "", "[agent]\nmax_side_sessions = 1\n")
	if cfg.SideSessionLimit() != 0 {
		t.Fatal("a cloned repo's reasonix.toml limited the user's side conversations")
	}
	if !slices.Contains(ignoredKeys(cfg), "agent.max_side_sessions") {
		t.Fatalf("ignored = %v, want agent.max_side_sessions reported", ignoredKeys(cfg))
	}
	cfg, _ = loadScoped(t, "[agent]\nmax_side_sessions = 5\n", "[agent]\nmax_side_sessions = 1\n")
	if cfg.SideSessionLimit() != 5 {
		t.Fatalf("the project file overrode the user's limit: %d", cfg.SideSessionLimit())
	}
}
