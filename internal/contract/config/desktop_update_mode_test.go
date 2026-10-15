package config

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestUpdateModeDefaultsToManualAndOnlyNotifyTurnsAnythingOn(t *testing.T) {
	for value, want := range map[string]string{"": "manual", "manual": "manual", "notify": "notify", " notify ": "notify", "download": "manual", "NOTIFY": "manual", "on": "manual"} {
		cfg := &Config{}
		cfg.Desktop.UpdateMode = value
		if got := cfg.DesktopUpdateMode(); got != want {
			t.Errorf("update_mode %q reads as %q, want %q", value, got, want)
		}
	}
	if (*Config)(nil).DesktopUpdateMode() != "manual" {
		t.Fatal("a missing config must read as manual")
	}
}

func TestSetUpdateModeRefusesWhatItDoesNotOffer(t *testing.T) {
	cfg := &Config{}
	if err := cfg.SetDesktopUpdateMode("download"); !errors.Is(err, ErrUpdateModeInvalid) || cfg.Desktop.UpdateMode != "" {
		t.Fatalf("err %v, stored %q; want a refusal that stores nothing", err, cfg.Desktop.UpdateMode)
	}
	if err := cfg.SetDesktopUpdateMode("notify"); err != nil || cfg.DesktopUpdateMode() != "notify" {
		t.Fatalf("notify: %v, reads %q", err, cfg.DesktopUpdateMode())
	}
}

func TestUpdateModeSurvivesARenderAndManualWritesNothing(t *testing.T) {
	cfg := Default()
	if strings.Contains(RenderTOMLForScope(cfg, RenderScopeUser), "update_mode") {
		t.Fatal("manual is the default and is not written")
	}
	_ = cfg.SetDesktopUpdateMode("notify")
	if !strings.Contains(RenderTOMLForScope(cfg, RenderScopeUser), `update_mode = "notify"`) {
		t.Fatal("notify must be written")
	}
}

func TestAProjectFileCannotTurnBackgroundUpdateChecksOn(t *testing.T) {
	cfg, _ := loadScoped(t, "", "[desktop]\nupdate_mode = \"notify\"\n")
	if cfg.DesktopUpdateMode() != "manual" || !slices.Contains(ignoredKeys(cfg), "desktop.update_mode") {
		t.Fatalf("mode %q, ignored %v; a project file must not choose it", cfg.DesktopUpdateMode(), ignoredKeys(cfg))
	}
	cfg, _ = loadScoped(t, "[desktop]\nupdate_mode = \"notify\"\n", "[desktop]\nupdate_mode = \"manual\"\n")
	if cfg.DesktopUpdateMode() != "notify" {
		t.Fatalf("a project file overrode the user's choice: %q", cfg.DesktopUpdateMode())
	}
}
