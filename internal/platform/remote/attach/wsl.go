package attach

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"reasonix/internal/contract/config"
	"reasonix/internal/platform/remote"
	"reasonix/internal/platform/remote/forward"
	"reasonix/internal/platform/remote/wslconn"
	"reasonix/internal/platform/wsl"
)

// ErrNoRemoteForward: a WSL distribution has no channel to publish a port of
// this machine into, so the one thing that would ask for it must not be sent.
var ErrNoRemoteForward = errors.New("attach: a WSL machine cannot be given a port of this one")

// wslLink is a WSL distribution as a Link. The distribution's loopback is
// reachable from here, so a "forward" is the address itself: nothing listens
// on this side and nothing has to be torn down.
type wslLink struct {
	*wslconn.Conn
	host string

	anchorStop context.CancelFunc
	anchorDone chan struct{}
	anchorIn   io.Writer // guarded by mu; nil while no session is up
	anchorUp   chan struct{}
	upOnce     sync.Once

	mu   sync.Mutex
	subs map[int]func(remote.StatusEvent)
	next int
	last remote.StatusEvent
	have bool
}

func newWSLLink(ctx context.Context, entry config.RemoteHostEntry, list wsl.Runner, run wsl.Command) (Link, error) {
	conn, err := wslconn.New(ctx, list, run, entry.Distro)
	if err != nil {
		return nil, err
	}
	return &wslLink{Conn: conn, host: entry.Name, subs: map[int]func(remote.StatusEvent){}, anchorUp: make(chan struct{})}, nil
}

func (l *wslLink) publish(ev remote.StatusEvent) {
	ev.Host, ev.At = l.host, time.Now()
	l.mu.Lock()
	l.last, l.have = ev, true
	fns := make([]func(remote.StatusEvent), 0, len(l.subs))
	for _, fn := range l.subs {
		fns = append(fns, fn)
	}
	l.mu.Unlock()
	for _, fn := range fns {
		fn(ev)
	}
}

// Start wakes the distribution: the first command is what boots it, and a
// distribution that cannot answer one is reported as the connect failing.
func (l *wslLink) Start(ctx context.Context) error {
	l.publish(remote.StatusEvent{Status: remote.StatusConnecting})
	res, err := l.Exec(ctx, "exit 0")
	if err == nil && res.ExitCode != 0 {
		err = fmt.Errorf("%w: %q answered with status %d", wsl.ErrUnsupported, l.Distro(), res.ExitCode)
	}
	if err != nil {
		l.publish(remote.StatusEvent{Status: remote.StatusStopped, Err: err})
		return err
	}
	l.publish(remote.StatusEvent{Status: remote.StatusConnected})
	anchorCtx, stop := context.WithCancel(context.WithoutCancel(ctx))
	l.anchorStop, l.anchorDone = stop, make(chan struct{})
	go l.holdAnchor(anchorCtx)
	select {
	case <-l.anchorUp:
	case <-ctx.Done():
		stop()
		<-l.anchorDone
		return ctx.Err()
	}
	return nil
}

// anchorRetry is how long a lost anchor waits before the distribution is asked
// for another.
var anchorRetry = 2 * time.Second

// holdAnchor keeps one shell session open and feeds it the commands that must
// outlive the call that sent them: WSL ends everything a wsl.exe session
// started when it ends, setsid and nohup included. A lost session is asked for
// again, and the kernel it held is gone.
func (l *wslLink) holdAnchor(ctx context.Context) {
	defer close(l.anchorDone)
	for attempt := 1; ctx.Err() == nil; attempt++ {
		pr, pw := io.Pipe()
		l.mu.Lock()
		l.anchorIn = pw
		l.mu.Unlock()
		l.upOnce.Do(func() { close(l.anchorUp) })
		sessionDone := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				_ = pw.Close()
			case <-sessionDone:
			}
		}()
		_, err := l.Run(ctx, pr, "/bin/sh")
		close(sessionDone)
		l.mu.Lock()
		l.anchorIn = nil
		l.mu.Unlock()
		_ = pw.Close()
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("%w: the session holding %q open ended", wsl.ErrUnsupported, l.Distro())
		}
		l.publish(remote.StatusEvent{Status: remote.StatusReconnecting, Attempt: attempt, Err: err})
		select {
		case <-ctx.Done():
			return
		case <-time.After(anchorRetry):
		}
		l.publish(remote.StatusEvent{Status: remote.StatusConnected, Attempt: attempt})
	}
}

// ErrNoSession: the session that launches long-lived processes is not up.
var ErrNoSession = errors.New("attach: the WSL session that keeps a kernel alive is not open")

// detachedPoll is how often a launched command is asked whether it finished.
const detachedPoll = 100 * time.Millisecond

// ExecDetached runs cmd inside the held session, so what it starts is not
// ended when this call returns. The command is written to a file and only its
// name is typed into the session, which keeps the script's text out of the
// session's input.
func (l *wslLink) ExecDetached(ctx context.Context, cmd string) (remote.ExecResult, error) {
	var raw [8]byte
	_, _ = rand.Read(raw[:])
	id := hex.EncodeToString(raw[:])
	const dir = "/tmp/.reasonix-link"
	put, err := l.Script(ctx, strings.NewReader(cmd), `mkdir -p -m 700 "$1" && cat > "$1/$2.sh"`, dir, id)
	if err != nil {
		return remote.ExecResult{}, err
	}
	if put.ExitCode != 0 {
		return remote.ExecResult{}, fmt.Errorf("attach: stage the launch command: exit %d: %s", put.ExitCode, strings.TrimSpace(string(put.Stderr)))
	}
	l.mu.Lock()
	in := l.anchorIn
	l.mu.Unlock()
	if in == nil {
		return remote.ExecResult{}, ErrNoSession
	}
	line := fmt.Sprintf("/bin/sh %[1]s/%[2]s.sh </dev/null >%[1]s/%[2]s.out 2>%[1]s/%[2]s.err; echo $? >%[1]s/%[2]s.rc\n", dir, id)
	wrote := make(chan error, 1)
	go func() { _, err := io.WriteString(in, line); wrote <- err }()
	select {
	case err := <-wrote:
		if err != nil {
			return remote.ExecResult{}, fmt.Errorf("%w: %w", ErrNoSession, err)
		}
	case <-ctx.Done():
		return remote.ExecResult{}, ctx.Err()
	}
	for {
		res, err := l.Script(ctx, nil, `[ -s "$1/$2.rc" ] || exit 44
code=$(cat "$1/$2.rc")
cat "$1/$2.out"
cat "$1/$2.err" >&2
rm -f "$1/$2.sh" "$1/$2.out" "$1/$2.err" "$1/$2.rc"
exit "$code"`, dir, id)
		if err != nil {
			return remote.ExecResult{}, err
		}
		if res.ExitCode != 44 {
			return remote.ExecResult{Stdout: res.Stdout, Stderr: res.Stderr, ExitCode: res.ExitCode}, nil
		}
		select {
		case <-ctx.Done():
			return remote.ExecResult{}, ctx.Err()
		case <-time.After(detachedPoll):
		}
	}
}

func (l *wslLink) Close() error {
	if l.anchorStop != nil {
		l.anchorStop()
		<-l.anchorDone
	}
	l.publish(remote.StatusEvent{Status: remote.StatusStopped})
	return nil
}

func (l *wslLink) Subscribe(fn func(remote.StatusEvent)) func() {
	l.mu.Lock()
	id := l.next
	l.next++
	l.subs[id] = fn
	last, have := l.last, l.have
	l.mu.Unlock()
	if have {
		fn(last)
	}
	return func() {
		l.mu.Lock()
		delete(l.subs, id)
		l.mu.Unlock()
	}
}

func (l *wslLink) Forwards() Forwards { return loopbackForwards{} }

// loopbackForwards reaches a port on the distribution's loopback directly.
type loopbackForwards struct{}

func (loopbackForwards) Add(spec forward.Spec) (string, error) {
	if spec.Direction != forward.Local {
		return "", ErrNoRemoteForward
	}
	return spec.TargetAddr, nil
}

func (loopbackForwards) List() []forward.Entry { return nil }

func (loopbackForwards) Remove(string) error { return nil }
