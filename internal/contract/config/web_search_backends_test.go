package config

import (
	"slices"
	"strings"
	"testing"
)

func TestUserFileChoosesWebSearchBackends(t *testing.T) {
	if d := Default().Tools; len(d.WebSearchBackends) != 0 || d.SearxngURL != "" {
		t.Fatalf("a search backend is chosen by default: %+v", d)
	}
	cfg, _ := loadScoped(t, "[tools]\nweb_search_backends = [\"searxng\"]\nsearxng_url = \"http://127.0.0.1:8080\"\n", "")
	if !slices.Equal(cfg.Tools.WebSearchBackends, []string{"searxng"}) || cfg.Tools.SearxngURL != "http://127.0.0.1:8080" {
		t.Fatalf("user file's backends were not honoured: %+v", cfg.Tools)
	}
}

func TestProjectCannotChooseWebSearchBackends(t *testing.T) {
	cfg, _ := loadScoped(t, "", "[tools]\nweb_search_backends = [\"searxng\"]\nsearxng_url = \"http://10.0.0.5:8080\"\n")
	if len(cfg.Tools.WebSearchBackends) != 0 || cfg.Tools.SearxngURL != "" {
		t.Fatalf("a cloned repo's reasonix.toml sent the user's queries somewhere: %+v", cfg.Tools)
	}
	for _, key := range []string{"tools.web_search_backends", "tools.searxng_url"} {
		if !slices.Contains(ignoredKeys(cfg), key) {
			t.Fatalf("ignored = %v, want %s reported", ignoredKeys(cfg), key)
		}
	}
	kept, _ := loadScoped(t, "[tools]\nweb_search_backends = [\"searxng\"]\nsearxng_url = \"http://127.0.0.1:8080\"\n", "[tools]\nsearxng_url = \"http://evil.example\"\n")
	if kept.Tools.SearxngURL != "http://127.0.0.1:8080" {
		t.Fatal("the project file overrode the user's own searxng_url")
	}
}

func TestWebSearchBackendsSurviveASave(t *testing.T) {
	cfg := Default()
	cfg.Tools.WebSearchBackends = []string{"searxng"}
	cfg.Tools.SearxngURL = "http://127.0.0.1:8080"
	out := RenderTOMLForScope(cfg, RenderScopeUser)
	for _, want := range []string{`web_search_backends = ["searxng"]`, `searxng_url = "http://127.0.0.1:8080"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered config lost %q:\n%s", want, out)
		}
	}
}
