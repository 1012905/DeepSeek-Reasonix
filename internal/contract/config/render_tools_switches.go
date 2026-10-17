package config

import (
	"fmt"
	"strings"
)

// mcpResourcesLine is the documented line in a full render of [tools].
func mcpResourcesLine(t ToolsConfig) string {
	line := fmt.Sprintf("mcp_resources = %v   # a tool that reads resources a connected MCP server declares; user file only\n", t.MCPResources)
	if len(t.WebSearchBackends) > 0 {
		line += fmt.Sprintf("web_search_backends = %s   # ordered chain web_search tries, free first; user file only\n", renderStringArray(t.WebSearchBackends))
	}
	if t.SearxngURL != "" {
		line += fmt.Sprintf("searxng_url = %q   # your own SearXNG instance with the json format enabled; user file only\n", t.SearxngURL)
	}
	return line
}

// renderToolsSwitches writes the [tools] switches a delta render carries when
// they are set.
func renderToolsSwitches(b *strings.Builder, t ToolsConfig) {
	if t.BrowserTools != nil {
		fmt.Fprintf(b, "browser_tools = %v\n", *t.BrowserTools)
	}
	if t.MCPResources {
		b.WriteString("mcp_resources = true\n")
	}
	if len(t.WebSearchBackends) > 0 {
		fmt.Fprintf(b, "web_search_backends = %s\n", renderStringArray(t.WebSearchBackends))
	}
	if t.SearxngURL != "" {
		fmt.Fprintf(b, "searxng_url = %q\n", t.SearxngURL)
	}
}
