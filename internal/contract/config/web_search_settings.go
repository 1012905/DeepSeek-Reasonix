package config

// WebSearchSettings is where web_search sends the user's queries. Every field
// is user file only: a project file cannot choose a destination for them or
// agree to a charge on the user's behalf.
type WebSearchSettings struct {
	// WebSearchBackends is the ordered chain web_search tries, free backends
	// first. Empty keeps the model-side search.
	WebSearchBackends []string `toml:"web_search_backends"`
	// SearxngURL is the user's own SearXNG instance, with its JSON format on.
	SearxngURL string `toml:"searxng_url"`
	// WebSearchPaidAllowed names the paid backends the user already agreed to be
	// charged for, so the card is not shown again.
	WebSearchPaidAllowed []string `toml:"web_search_paid_allowed"`
}
