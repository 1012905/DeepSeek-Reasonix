package sessionstore

import "strings"

// planModeMarkerPrefix opens the plan-mode contract the host prepends to a
// planning turn; the person's words follow the closing bracket.
const planModeMarkerPrefix = "[Plan mode"

// UserSaidText is the one definition of what a persisted user turn says: the
// host's prepended blocks and the plan-mode marker are not the person's words.
// Session history's index and its snippet both call it, so a hit and its
// snippet cannot disagree.
func UserSaidText(content string) string {
	trimmed := strings.TrimSpace(DropLeadingTransientBlocks(content))
	if rest, ok := strings.CutPrefix(trimmed, planModeMarkerPrefix); ok {
		if _, after, ok := strings.Cut(rest, "]"); ok {
			return strings.TrimSpace(after)
		}
	}
	return trimmed
}
