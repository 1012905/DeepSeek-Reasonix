package serve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

// dispatchBook remembers which pane a sender opened in which project, so a
// second task to the same project goes to the session already waiting instead
// of opening another.
type dispatchBook struct {
	mu       sync.Mutex
	sessions map[dispatchKey]string
	open     func(context.Context, OpenRequest) (*Runtime, error) // nil: Hub.Open
}

type dispatchKey struct{ from, root string }

func (b *dispatchBook) get(from, root string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sessions[dispatchKey{from, root}]
}

func (b *dispatchBook) set(from, root, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sessions == nil {
		b.sessions = map[dispatchKey]string{}
	}
	b.sessions[dispatchKey{from, root}] = id
}

// dispatcherFor is what a pane's controller calls to hand a task to another of
// the user's projects. The controller has already checked the user's switch.
func (h *Hub) dispatcherFor(from *Runtime) control.ProjectDispatcher {
	return func(ctx context.Context, project, text string) (control.DispatchResult, error) {
		return h.dispatchTo(ctx, from, project, text)
	}
}

func (h *Hub) dispatchTo(ctx context.Context, from *Runtime, project, text string) (control.DispatchResult, error) {
	root, err := resolveDispatchProject(project, Workspaces())
	if err != nil {
		return control.DispatchResult{}, err
	}
	target := h.runtimeByID(h.dispatch.get(from.ID, root))
	if target == nil {
		if target, err = h.openDispatchSession(ctx, from, root); err != nil {
			return control.DispatchResult{}, err
		}
	}
	targetCtrl, ok := target.Server.Controller().(*control.Controller)
	sender, senderOK := from.Server.Controller().(*control.Controller)
	if !ok || !senderOK {
		return control.DispatchResult{}, control.ErrDispatchUnavailable
	}
	key := targetCtrl.PeerKey()
	if key == "" {
		return control.DispatchResult{}, fmt.Errorf("%w: the new session did not publish itself", control.ErrDispatchFailed)
	}
	receipt, err := sender.SendPeerMessage(key, text)
	if err != nil {
		return control.DispatchResult{}, err
	}
	return control.DispatchResult{Project: root, Key: key, ID: receipt.ID}, nil
}

// openDispatchSession opens a fresh pane in root and gives it the file the
// sender will write to. A folder the user has not trusted is not opened for
// someone else's task: the pane is closed again and the dispatch refused.
func (h *Hub) openDispatchSession(ctx context.Context, from *Runtime, root string) (*Runtime, error) {
	open := h.dispatch.open
	if open == nil {
		open = h.Open
	}
	target, err := open(ctx, OpenRequest{Root: root})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", control.ErrDispatchFailed, err)
	}
	ctrl, ok := target.Server.Controller().(*control.Controller)
	if !ok {
		_ = h.Close(target.ID)
		return nil, control.ErrDispatchUnavailable
	}
	if ctrl.WorkspaceTrust() != config.WorkspaceTrusted {
		_ = h.Close(target.ID)
		return nil, control.ErrDispatchUntrusted
	}
	ctrl.MarkOpenedByDispatch()
	ctrl.SetFreshSessionPath(sessionstore.NewSessionPath(SessionDirFor(root), "dispatch"))
	h.dispatch.set(from.ID, root, target.ID)
	return target, nil
}

func (h *Hub) runtimeByID(id string) *Runtime {
	if id == "" {
		return nil
	}
	for _, rt := range h.localRuntimes() {
		if rt.ID == id {
			return rt
		}
	}
	return nil
}

// resolveDispatchProject maps what the model wrote onto one of the user's own
// projects: an exact path, else a folder name. Two projects with one name, or
// none, is refused; a folder outside the list never resolves.
func resolveDispatchProject(name string, projects []string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", control.ErrDispatchUnknownProject
	}
	if filepath.IsAbs(name) {
		want := filepath.Clean(name)
		for _, p := range projects {
			if filepath.Clean(p) == want && dirExists(p) {
				return p, nil
			}
		}
		return "", control.ErrDispatchUnknownProject
	}
	var found []string
	for _, p := range projects {
		if strings.EqualFold(filepath.Base(p), name) && dirExists(p) {
			found = append(found, p)
		}
	}
	switch len(found) {
	case 0:
		return "", control.ErrDispatchUnknownProject
	case 1:
		return found[0], nil
	}
	return "", control.ErrDispatchAmbiguous
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
