package control

import "reasonix/internal/contract/config"

// WriteLeaseSettings is how this machine's sessions hold the workspace write
// lease against one another, as the user file records it. It is the user's
// alone: a project file cannot set it, so there is no second effective value.
type WriteLeaseSettings struct {
	Optimistic bool   `json:"optimistic"`
	Path       string `json:"path"`
}

// WriteLeaseSettings reads [agent] write_lease from the user file.
func (c *Controller) WriteLeaseSettings() WriteLeaseSettings {
	path := config.UserConfigPath()
	return WriteLeaseSettings{Optimistic: config.LoadForEdit(path).WriteLeaseOptimistic(), Path: path}
}

// SaveWriteLeaseSettings persists the choice to the user file. A session keeps
// the lease it was built with, and a rebuild adopts it, so the choice reaches
// the sessions opened after it, not the ones already running.
func (c *Controller) SaveWriteLeaseSettings(optimistic bool) error {
	unlock := config.LockUserConfigEdits()
	defer unlock()
	path := config.UserConfigPath()
	cfg := config.LoadForEdit(path)
	cfg.SetWriteLease(optimistic)
	return cfg.SaveTo(path)
}
