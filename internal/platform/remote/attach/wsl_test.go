package attach

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/platform/remote"
	"reasonix/internal/platform/remote/forward"
	"reasonix/internal/platform/wsl"
)

// wslMachine is a fake wsl.exe: it lists one distribution and records the
// commands it is asked to run.
type wslMachine struct {
	t       *testing.T
	version int
	mu      sync.Mutex
	calls   [][]string
	// anchor decides how the long-lived session behaves; nil blocks until cancelled.
	anchor func(ctx context.Context, stdin io.Reader) (wsl.Result, error)
	// script answers every other command; nil answers success with no output.
	script func(args []string, stdin io.Reader) (wsl.Result, error)
	// typed collects what the session was sent.
	typed   strings.Builder
	typedMu sync.Mutex
	// started gets one value each time the long-lived session is launched.
	started chan struct{}
}

func (m *wslMachine) list(context.Context, ...string) ([]byte, error) {
	v := m.version
	if v == 0 {
		v = 2
	}
	return []byte("  NAME      STATE   VERSION\n* Ubuntu    Running " + string(rune('0'+v)) + "\n"), nil
}

func (m *wslMachine) run(ctx context.Context, stdin io.Reader, args ...string) (wsl.Result, error) {
	m.mu.Lock()
	m.calls = append(m.calls, slices.Clone(args))
	m.mu.Unlock()
	if slices.Equal(args, []string{"-d", "Ubuntu", "--exec", "/bin/sh"}) {
		if m.started != nil {
			m.started <- struct{}{}
		}
		if m.anchor != nil {
			return m.anchor(ctx, stdin)
		}
		return m.session(ctx, stdin)
	}
	if m.script != nil {
		return m.script(args, stdin)
	}
	return wsl.Result{}, nil
}

// session is a shell that reads its commands until it is cancelled or its
// input closes, the way /bin/sh does.
func (m *wslMachine) session(ctx context.Context, stdin io.Reader) (wsl.Result, error) {
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdin.Read(buf)
			m.typedMu.Lock()
			m.typed.Write(buf[:n])
			m.typedMu.Unlock()
			if err != nil {
				close(done)
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		return wsl.Result{}, ctx.Err()
	case <-done:
		return wsl.Result{}, nil
	}
}

func (m *wslMachine) link(entry config.RemoteHostEntry) (Link, error) {
	return newWSLLink(m.t.Context(), entry, m.list, m.run)
}

func wslEntry() config.RemoteHostEntry {
	return config.RemoteHostEntry{Name: "dev", Kind: "wsl", Distro: "Ubuntu"}
}

func TestAWSLLinkIsBoundToAnInstalledWSL2Distribution(t *testing.T) {
	m := &wslMachine{t: t}
	if _, err := m.link(config.RemoteHostEntry{Name: "x", Kind: "wsl", Distro: "Nope"}); !errors.Is(err, wsl.ErrNoDistro) {
		t.Fatalf("unknown distribution = %v", err)
	}
	m.version = 1
	if _, err := m.link(wslEntry()); !errors.Is(err, wsl.ErrUnsupported) {
		t.Fatalf("WSL1 = %v, want ErrUnsupported", err)
	}
	m.version = 2
	if _, err := m.link(wslEntry()); err != nil {
		t.Fatal(err)
	}
}

func TestStartingAWSLLinkHoldsOneSessionOpenUntilClose(t *testing.T) {
	m := &wslMachine{t: t, started: make(chan struct{}, 4)}
	l, err := m.link(wslEntry())
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []remote.Status
	l.Subscribe(func(ev remote.StatusEvent) { mu.Lock(); seen = append(seen, ev.Status); mu.Unlock() })
	if err := l.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	<-m.started
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(seen, []remote.Status{remote.StatusConnecting, remote.StatusConnected, remote.StatusStopped}) {
		t.Fatalf("states = %v", seen)
	}
	var anchors int
	for _, c := range m.calls {
		if slices.Equal(c, []string{"-d", "Ubuntu", "--exec", "/bin/sh"}) {
			anchors++
		}
	}
	if anchors != 1 {
		t.Fatalf("%d sessions were held open, want 1", anchors)
	}
}

func TestALostSessionIsReportedAndAskedForAgain(t *testing.T) {
	defer func(d time.Duration) { anchorRetry = d }(anchorRetry)
	anchorRetry = time.Millisecond
	var starts int
	var mu sync.Mutex
	m := &wslMachine{t: t, started: make(chan struct{}, 4), anchor: func(ctx context.Context, _ io.Reader) (wsl.Result, error) {
		mu.Lock()
		starts++
		n := starts
		mu.Unlock()
		if n == 1 {
			return wsl.Result{}, nil
		}
		<-ctx.Done()
		return wsl.Result{}, ctx.Err()
	}}
	l, _ := m.link(wslEntry())
	events := make(chan remote.StatusEvent, 16)
	l.Subscribe(func(ev remote.StatusEvent) { events <- ev })
	if err := l.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	var sawLost bool
	for !sawLost {
		select {
		case ev := <-events:
			sawLost = ev.Status == remote.StatusReconnecting && ev.Err != nil && ev.Attempt == 1
		case <-deadline:
			t.Fatal("a session that ended was not reported")
		}
	}
	<-m.started
	<-m.started
	_ = l.Close()
	mu.Lock()
	defer mu.Unlock()
	if starts != 2 {
		t.Fatalf("the session was started %d times, want 2", starts)
	}
}

func TestAWSLLinkReachesTheKernelDirectlyAndRefusesAnInboundPort(t *testing.T) {
	f := loopbackForwards{}
	addr, err := f.Add(forward.Spec{Direction: forward.Local, BindAddr: "127.0.0.1:0", TargetAddr: "127.0.0.1:4242"})
	if err != nil || addr != "127.0.0.1:4242" {
		t.Fatalf("Add local = %q, %v", addr, err)
	}
	if _, err := f.Add(forward.Spec{Direction: forward.Remote, BindAddr: "127.0.0.1:0", TargetAddr: "127.0.0.1:1"}); !errors.Is(err, ErrNoRemoteForward) {
		t.Fatalf("Add remote = %v, want ErrNoRemoteForward", err)
	}
	if len(f.List()) != 0 || f.Remove("x") != nil {
		t.Fatal("a direct route has nothing to list or remove")
	}
}

// A WSL entry resolves providers on the distribution: a broker configured on
// this pool must not be published to it.
func TestAPoolOpensAWSLEntryThroughTheWSLLinkAndSendsNoBroker(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	if err := config.EditUserConfigWithCredentials(func(c *config.Config) ([]config.CredentialChange, error) {
		return nil, c.UpsertRemoteHost(wslEntry())
	}); err != nil {
		t.Fatal(err)
	}
	m := &wslMachine{t: t}
	var opened int
	p := NewPool(t.Context(), Options{
		Broker: Broker{Addr: "127.0.0.1:9", Token: "t"},
		Dial: func(string, Prompts) (*remote.Client, error) {
			t.Error("an SSH dial was made for a WSL entry")
			return nil, errors.New("no")
		},
		DialWSL: func(e config.RemoteHostEntry) (Link, error) { opened++; return m.link(e) },
	})
	defer p.Close()
	l, dialer := p.holdLink("dev")
	if !dialer {
		t.Fatal("first hold must dial")
	}
	p.dial(l, Call{})
	if err := l.wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if opened != 1 || p.brokers(l) {
		t.Fatalf("opened = %d, brokers = %v", opened, p.brokers(l))
	}
}

// What a launch starts must belong to the held session, not to the call that
// sent it: the command is staged in a file, only its name is typed into the
// session, and the call returns what the command printed.
func TestADetachedCommandRunsInTheHeldSessionAndReturnsItsOutput(t *testing.T) {
	var staged string
	var polls int
	m := &wslMachine{t: t}
	m.script = func(args []string, stdin io.Reader) (wsl.Result, error) {
		body := args[5]
		switch {
		case strings.Contains(body, `cat > "$1/$2.sh"`):
			b, _ := io.ReadAll(stdin)
			staged = string(b)
			return wsl.Result{}, nil
		case strings.Contains(body, `.rc`):
			polls++
			if polls < 3 {
				return wsl.Result{ExitCode: 44}, nil
			}
			return wsl.Result{Stdout: []byte("12345\n")}, nil
		}
		return wsl.Result{}, nil
	}
	l, _ := m.link(wslEntry())
	d := l.(interface {
		ExecDetached(context.Context, string) (remote.ExecResult, error)
	})
	if _, err := d.ExecDetached(t.Context(), "x"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("before the session is up: %v, want ErrNoSession", err)
	}
	if err := l.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	res, err := d.ExecDetached(t.Context(), "nohup serve & echo $!")
	if err != nil || string(res.Stdout) != "12345\n" || res.ExitCode != 0 {
		t.Fatalf("ExecDetached = %+v, %v", res, err)
	}
	if staged != "nohup serve & echo $!" || polls != 3 {
		t.Fatalf("staged = %q, polls = %d", staged, polls)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		m.typedMu.Lock()
		typed := m.typed.String()
		m.typedMu.Unlock()
		if strings.Contains(typed, "/tmp/.reasonix-link/") && strings.HasSuffix(typed, "\n") && !strings.Contains(typed, "nohup") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the session was sent %q, want a line naming the staged file and none of its text", typed)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestADistributionWithoutPsIsRefusedBeforeAnythingIsLaunched(t *testing.T) {
	m := &wslMachine{t: t, script: func(args []string, _ io.Reader) (wsl.Result, error) {
		return wsl.Result{Stdout: []byte("ps\n"), ExitCode: exitMissingTool}, nil
	}}
	l, _ := m.link(wslEntry())
	err := l.Start(t.Context())
	if !errors.Is(err, wsl.ErrUnsupported) || !strings.Contains(err.Error(), "no ps") {
		t.Fatalf("Start = %v, want ErrUnsupported naming ps", err)
	}
	for _, c := range m.calls {
		if slices.Equal(c, []string{"-d", "Ubuntu", "--exec", "/bin/sh"}) {
			t.Fatal("a session was opened on a distribution that cannot run a kernel")
		}
	}
}

func TestAWindowsWorkspaceIsTranslatedByTheDistributionBeforeTheKernelIsAsked(t *testing.T) {
	m := &wslMachine{t: t, script: func(args []string, _ io.Reader) (wsl.Result, error) {
		if len(args) == 6 && args[3] == "wslpath" && args[5] == `C:\src\app` {
			return wsl.Result{Stdout: []byte("/mnt/c/src/app\n")}, nil
		}
		return wsl.Result{}, nil
	}}
	l, _ := m.link(wslEntry())
	w := l.(workspaceMapper)
	for in, want := range map[string]string{`C:\src\app`: "/mnt/c/src/app", "/home/dev/app": "/home/dev/app", "~/app": "~/app"} {
		if got, err := w.Workspace(t.Context(), in); err != nil || got != want {
			t.Errorf("Workspace(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := w.Workspace(t.Context(), `Z:\none`); err == nil {
		t.Fatal("a drive the distribution does not have was accepted")
	}
}
