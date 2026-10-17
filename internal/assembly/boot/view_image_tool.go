package boot

import (
	"slices"

	"reasonix/internal/contract/config"
	"reasonix/internal/contract/tool"
	"reasonix/internal/tools/builtin"
)

// addViewImage registers view_image for a session whose model reads images. A
// session on a text-only model is not offered a tool it cannot use, and its tool
// block (and so its cached prompt prefix) is unchanged.
func addViewImage(reg *tool.Registry, cfg *config.Config, entry *config.ProviderEntry, env toolEnvironment, root string) {
	if !config.EffectiveVision(entry) {
		return
	}
	if len(cfg.Tools.Enabled) > 0 && !slices.Contains(cfg.Tools.Enabled, "view_image") {
		return
	}
	reg.Add(builtin.Workspace{
		Dir: root, WriteRoots: env.writeRoots, ReadRoots: env.readRoots, ForbidReadRoots: env.forbidReadRoots, ReadPaths: env.readPaths,
	}.ViewImage())
}
