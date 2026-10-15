package config

import (
	"errors"
	"strings"
)

// The update modes, from least to most automatic. Manual is the default and
// what any other value reads as: a misspelt or hand-edited mode must never turn
// background work on.
const (
	UpdateModeManual = "manual"
	UpdateModeNotify = "notify"
)

// ErrUpdateModeInvalid is a mode this build does not offer.
var ErrUpdateModeInvalid = errors.New("config: unknown update mode")

// DesktopUpdateMode is how the desktop behaves about new releases: manual does
// nothing on its own, notify reads the version catalog in the background and
// says so when something newer is published. It installs nothing either way.
func (c *Config) DesktopUpdateMode() string {
	if c != nil && strings.TrimSpace(c.Desktop.UpdateMode) == UpdateModeNotify {
		return UpdateModeNotify
	}
	return UpdateModeManual
}

// SetDesktopUpdateMode records the user's choice. An unknown mode is refused
// rather than stored, so the file never holds a value that reads as something
// the user did not pick.
func (c *Config) SetDesktopUpdateMode(mode string) error {
	switch m := strings.TrimSpace(mode); m {
	case UpdateModeManual, UpdateModeNotify:
		c.Desktop.UpdateMode = m
		return nil
	}
	return ErrUpdateModeInvalid
}
