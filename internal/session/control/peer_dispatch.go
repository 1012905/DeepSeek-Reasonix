package control

import (
	"context"
	"errors"
	"sync/atomic"

	"reasonix/internal/contract/config"
	"reasonix/internal/contract/tool"
)

// Each is an identity the dispatch tool reports; none is matched by its words.
var (
	ErrDispatchUnavailable    = errors.New("this session cannot open sessions in other projects")
	ErrDispatchUnknownProject = errors.New("no project of yours answers to that name")
	ErrDispatchAmbiguous      = errors.New("more than one project answers to that name")
	ErrDispatchNested         = errors.New("a session opened by a dispatch cannot dispatch further")
	ErrDispatchUntrusted      = errors.New("that project is not one you have trusted")
	ErrDispatchFailed         = errors.New("could not open a session in that project")
)

// DispatchResult is what a dispatch hands back: where the task went.
type DispatchResult struct {
	Project string
	Key     string
	ID      string
}

// ProjectDispatcher opens, or reuses, a session in another project of the user
// and writes the task into it. The host that owns the panes supplies it.
type ProjectDispatcher func(ctx context.Context, project, text string) (DispatchResult, error)

// dispatchState is the controller's side of cross-project dispatch.
type dispatchState struct {
	hook   atomic.Pointer[ProjectDispatcher]
	nested atomic.Bool
}

// SetProjectDispatcher gives the dispatch tool somewhere to go; a controller
// without one (a terminal session, a test) answers ErrDispatchUnavailable.
func (c *Controller) SetProjectDispatcher(d ProjectDispatcher) {
	if d == nil {
		c.dispatch.hook.Store(nil)
		return
	}
	c.dispatch.hook.Store(&d)
}

// MarkOpenedByDispatch records that another session opened this one to hand it
// a task. Such a session cannot dispatch in turn, which bounds the fan-out at
// one level.
func (c *Controller) MarkOpenedByDispatch() { c.dispatch.nested.Store(true) }

// OpenedByDispatch reports whether MarkOpenedByDispatch was called.
func (c *Controller) OpenedByDispatch() bool { return c.dispatch.nested.Load() }

// PeerKey is the address other sessions write to this one at, empty until the
// session has a file.
func (c *Controller) PeerKey() string {
	p := c.peers
	if p == nil {
		return ""
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.key
}

// DispatchToProject hands a task to a session in another of the user's projects.
func (c *Controller) DispatchToProject(ctx context.Context, project, text string) (DispatchResult, error) {
	if !config.LoadForEdit(config.UserConfigPath()).PeersDispatch() {
		return DispatchResult{}, ErrDispatchUnavailable
	}
	if c.OpenedByDispatch() {
		return DispatchResult{}, ErrDispatchNested
	}
	hook := c.dispatch.hook.Load()
	if hook == nil {
		return DispatchResult{}, ErrDispatchUnavailable
	}
	return (*hook)(ctx, project, text)
}

func (d peerDirectory) DispatchToProject(ctx context.Context, project, text string) (tool.DispatchReceipt, error) {
	r, err := d.c.DispatchToProject(ctx, project, text)
	if err != nil {
		return tool.DispatchReceipt{}, fmtPeerError(err)
	}
	return tool.DispatchReceipt{Project: r.Project, Key: r.Key, ID: r.ID}, nil
}

// RefreshPeers reads this session's spool now rather than at the next tick, so
// a message just sent to it can be seen at once.
func (c *Controller) RefreshPeers() { c.peerIngest() }
