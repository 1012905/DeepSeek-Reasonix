package tool

import "sync"

// PeerView is another session on this machine that can be written to.
type PeerView struct {
	Key     string
	Title   string
	Project string
}

// PeerReceipt says a message was handed to the receiver's spool, not read.
type PeerReceipt struct {
	ID string
	To PeerView
}

// PeerDirectory is what the peer tools ask of the session that owns them.
type PeerDirectory interface {
	Peers() []PeerView
	SendPeerMessage(to, text string) (PeerReceipt, error)
}

// PeerLink joins the peer tools, built with the registry, to the controller
// that exists only afterwards. Until the controller binds it the tools answer
// that no session is listening.
type PeerLink struct {
	mu  sync.RWMutex
	dir PeerDirectory
}

func (l *PeerLink) Bind(d PeerDirectory) {
	l.mu.Lock()
	l.dir = d
	l.mu.Unlock()
}

func (l *PeerLink) Directory() (PeerDirectory, bool) {
	if l == nil {
		return nil, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.dir, l.dir != nil
}
