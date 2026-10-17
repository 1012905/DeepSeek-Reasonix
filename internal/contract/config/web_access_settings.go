package config

// WebAccessSettings is where web_search and web_fetch send what the user and the
// model ask for. Every field is user file only: a project file cannot choose a
// destination for them or agree to a charge on the user's behalf.
type WebAccessSettings struct {
	// WebSearchBackends is the ordered chain web_search tries, free backends
	// first. Empty keeps the model-side search.
	WebSearchBackends []string `toml:"web_search_backends"`
	// SearxngURL is the user's own SearXNG instance, with its JSON format on.
	SearxngURL string `toml:"searxng_url"`
	// WebSearchPaidAllowed names the paid search backends the user already agreed
	// to be charged for, so the card is not shown again.
	WebSearchPaidAllowed []string `toml:"web_search_paid_allowed"`
	// WebFetchBackends is the ordered chain web_fetch tries: "local" is the free
	// fetch from this machine, "deepseek_api" the paid one. Empty keeps the
	// local fetch alone.
	WebFetchBackends []string `toml:"web_fetch_backends"`
	// WebFetchPaidAllowed is WebSearchPaidAllowed for fetching.
	WebFetchPaidAllowed []string `toml:"web_fetch_paid_allowed"`
}

// any reports whether the user set anything here.
func (w WebAccessSettings) any() bool {
	return len(w.WebSearchBackends) > 0 || w.SearxngURL != "" || len(w.WebSearchPaidAllowed) > 0 ||
		len(w.WebFetchBackends) > 0 || len(w.WebFetchPaidAllowed) > 0
}
