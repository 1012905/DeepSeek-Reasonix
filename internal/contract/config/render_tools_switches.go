package config

import (
	"fmt"
	"strings"
)

// mcpResourcesLine is the documented line in a full render of [tools].
func mcpResourcesLine(t ToolsConfig) string {
	return fmt.Sprintf("mcp_resources = %v   # a tool that reads resources a connected MCP server declares; user file only\n", t.MCPResources)
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
}
