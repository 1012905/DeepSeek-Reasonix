package boot

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"reasonix/internal/base/netclient"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
	"reasonix/internal/contract/tool"
	"reasonix/internal/tools/websearch"
)

const searxngTimeout = 20 * time.Second

// addSearchChain registers web_search over the user's ordered backends. A
// backend that cannot be built is reported and left out; with none left the
// tool is not registered at all, so the model never sees a search that cannot run.
func addSearchChain(reg *tool.Registry, cfg *config.Config, proxy netclient.ProxySpec, sink event.Sink) {
	if cfg.Environment.Offline || (len(cfg.Tools.Enabled) > 0 && !slices.Contains(cfg.Tools.Enabled, "web_search")) {
		return
	}
	var backends []websearch.Backend
	for _, name := range cfg.Tools.WebSearchBackends {
		backend, err := buildSearchBackend(name, cfg, proxy)
		if err != nil {
			report(sink, event.Event{Level: event.LevelWarn, Code: event.NoticeCodeWebSearchBackendUnusable,
				Text: "A configured web search backend cannot be used and was skipped: " + err.Error(), Detail: name})
			continue
		}
		backends = append(backends, backend)
	}
	if len(backends) == 0 {
		return
	}
	reg.Add(&websearch.Chain{
		Backends: backends,
		OnFallback: func(f websearch.Fallback) {
			next := f.Next
			if next == "" {
				next = "none"
			}
			report(sink, event.Event{Level: event.LevelWarn, Code: event.NoticeCodeWebSearchBackendFallback,
				Text:   fmt.Sprintf("Web search backend %s did not answer (%s); trying %s.", f.Backend, f.Kind, next),
				Detail: fmt.Sprintf("%s|%s|%d|%s", f.Backend, f.Kind, f.Status, next)})
		},
	})
}

func buildSearchBackend(name string, cfg *config.Config, proxy netclient.ProxySpec) (websearch.Backend, error) {
	switch name {
	case "searxng":
		client, err := netclient.NewHTTPClient(proxy, netclient.TransportOptions{})
		if err != nil {
			return nil, err
		}
		client.Timeout = searxngTimeout
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		return websearch.NewSearxng(cfg.Tools.SearxngURL, client)
	default:
		return nil, fmt.Errorf("unknown backend %q", name)
	}
}
