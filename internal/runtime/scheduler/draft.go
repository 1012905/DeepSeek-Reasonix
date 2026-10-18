package scheduler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/state/schedule"
)

// Typed reasons a draft cannot become a request; a frontend maps them by
// identity and never by wording.
var (
	ErrDraftTrigger   = errors.New("schedule draft: give exactly one of an interval and a time")
	ErrDraftWorkspace = errors.New("schedule draft: the folder does not exist")
	ErrDraftModel     = errors.New("schedule draft: the model is not configured")
)

// Draft is what a person typed or picked, before the host normalises it: the
// one shape every frontend builds a schedule from.
type Draft struct {
	Label     string
	EverySec  int64
	At        time.Time
	Workspace string
	Prompt    string
	// Model is a provider/model reference; empty is the user's default.
	Model string
}

// Request normalises a draft into the store's request: the folder is made
// absolute and resolved, and the model is pinned from the user's configuration.
// The returned request is marked as a human's; a frontend only calls this for
// one.
func (d Draft) Request() (schedule.CreateRequest, error) {
	req := schedule.CreateRequest{Label: strings.TrimSpace(d.Label), Prompt: strings.TrimSpace(d.Prompt), ConfirmedBy: schedule.ConfirmedByHuman}
	if (d.EverySec > 0) == !d.At.IsZero() {
		return req, ErrDraftTrigger
	}
	if d.EverySec > 0 {
		req.Trigger = schedule.Trigger{Kind: schedule.TriggerEvery, EverySec: d.EverySec}
	} else {
		req.Trigger = schedule.Trigger{Kind: schedule.TriggerAt, At: d.At}
	}
	ws := d.Workspace
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			return req, err
		}
		ws = wd
	}
	abs, err := filepath.Abs(ws)
	if err != nil {
		return req, err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return req, fmt.Errorf("%w: %q", ErrDraftWorkspace, ws)
	}
	req.Target = schedule.Target{Workspace: abs}
	model, err := PinModel(d.Model, abs)
	if err != nil {
		return req, fmt.Errorf("%w: %w", ErrDraftModel, err)
	}
	req.Model = model
	return req, nil
}
