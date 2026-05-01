// Package egress is the per-invocation HTTP CONNECT proxy that coily starts
// for the duration of a wrapped subprocess. The child inherits HTTPS_PROXY /
// HTTP_PROXY pointing at the proxy. The proxy logs every CONNECT, joins the
// rows back to the parent invocation's audit row, and (per-binary) either
// enforces a default-deny allowlist or observes silently.
//
// Two modes:
//
//   - ModeEnforce: per-binary allowlist pinned in code. Denied CONNECTs
//     return 403 and the row is marked decision=deny. Used for the package-
//     manager wrappers (brew, npm, pip, cargo, ...).
//   - ModeObserve: no allowlist. Every CONNECT is forwarded and logged.
//     Used for aws/gh/kubectl/docker/tailscale (issue #33).
//
// CONNECT-only. No TLS interception, no CA install. Hostnames come from
// the CONNECT request line (host:port). Plaintext HTTP requests through the
// proxy are rejected with 405 - any tool that relies on plaintext for
// package fetching is already a bug.
package egress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
)

// Mode selects allowlist enforcement vs observe-only logging.
type Mode int

const (
	// ModeEnforce denies any host not on the allowlist.
	ModeEnforce Mode = iota
	// ModeObserve forwards every CONNECT and logs it. Allowlist is ignored.
	ModeObserve
)

// Proxy is a single-use CONNECT proxy. New, Start, run the wrapped
// subprocess, Stop. Not safe for reuse across invocations; build a fresh
// Proxy per coily verb invocation.
type Proxy struct {
	allowlist map[string]bool
	mode      Mode

	listener net.Listener
	server   *http.Server

	mu   sync.Mutex
	rows map[string]*hostStat

	// inflight tracks active CONNECT tunnels. http.Server does not track
	// hijacked connections, so we count them ourselves and wait on Stop
	// before draining rows.
	inflight sync.WaitGroup

	// Now is overridable for tests.
	Now func() time.Time
}

type hostStat struct {
	decision   string
	bytesUp    int64
	bytesDown  int64
	durationMS int64
}

// New returns a Proxy with the given allowlist and mode. The allowlist is
// ignored when mode is ModeObserve. Hostname match is exact; suffix matching
// (e.g. "*.crates.io") is a Phase 2 follow-up.
func New(allowlist []string, mode Mode) *Proxy {
	set := make(map[string]bool, len(allowlist))
	for _, h := range allowlist {
		set[strings.ToLower(h)] = true
	}
	return &Proxy{
		allowlist: set,
		mode:      mode,
		rows:      make(map[string]*hostStat),
		Now:       time.Now,
	}
}

// Start binds to 127.0.0.1:0 and starts serving. Returns the http://host:port
// URL the child should use as HTTPS_PROXY/HTTP_PROXY.
func (p *Proxy) Start(_ context.Context) (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("egress: listen: %w", err)
	}
	p.listener = ln
	p.server = &http.Server{
		Handler:           http.HandlerFunc(p.handle),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = p.server.Serve(ln)
	}()
	return "http://" + ln.Addr().String(), nil
}

// Stop shuts the listener and returns the collected rows, one per host.
// Safe to call once. After Stop the Proxy is not reusable.
func (p *Proxy) Stop() []audit.EgressRow {
	if p.listener != nil {
		_ = p.listener.Close()
	}
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.server.Shutdown(ctx)
	}
	// Wait for any in-flight CONNECT tunnels to finish recording. http.Server
	// does not track hijacked connections so Shutdown returns immediately.
	p.inflight.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]audit.EgressRow, 0, len(p.rows))
	for host, s := range p.rows {
		out = append(out, audit.EgressRow{
			Host:       host,
			Decision:   s.decision,
			BytesUp:    s.bytesUp,
			BytesDown:  s.bytesDown,
			DurationMS: s.durationMS,
		})
	}
	// Stable order so audit rows diff cleanly between runs.
	sort.Slice(out, func(i, j int) bool { return out[i].Host < out[j].Host })
	return out
}

func (p *Proxy) allowed(host string) bool {
	if p.mode == ModeObserve {
		return true
	}
	return p.allowlist[strings.ToLower(host)]
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	p.inflight.Add(1)
	defer p.inflight.Done()
	if r.Method != http.MethodConnect {
		http.Error(w, "egress: only CONNECT is supported", http.StatusMethodNotAllowed)
		return
	}
	hostport := r.Host
	if hostport == "" {
		hostport = r.RequestURI
	}
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.ToLower(host)

	if !p.allowed(host) {
		p.record(host, audit.EgressDeny, 0, 0, 0)
		http.Error(w, "egress: host denied by coily allowlist", http.StatusForbidden)
		return
	}

	start := p.now()
	// hostport is taken from the CONNECT request line by design - this
	// process IS the proxy and the whole point is to dial what the client
	// asked for, after the allowlist gate above.
	upstream, err := net.DialTimeout("tcp", hostport, 10*time.Second) //nolint:gosec // G107/G704: intentional proxy dial
	if err != nil {
		p.record(host, audit.EgressDeny, 0, 0, p.now().Sub(start).Milliseconds())
		http.Error(w, "egress: upstream unreachable", http.StatusBadGateway)
		return
	}
	client, ok := hijack(w)
	if !ok {
		_ = upstream.Close()
		return
	}
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = upstream.Close()
		_ = client.Close()
		p.record(host, audit.EgressDeny, 0, 0, p.now().Sub(start).Milliseconds())
		return
	}
	up, down := tunnel(client, upstream)
	p.record(host, audit.EgressAllow, up, down, p.now().Sub(start).Milliseconds())
}

// hijack returns the raw client conn or, on failure, writes the error
// response and returns ok=false.
func hijack(w http.ResponseWriter) (net.Conn, bool) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "egress: hijack unsupported", http.StatusInternalServerError)
		return nil, false
	}
	c, _, err := hj.Hijack()
	if err != nil {
		return nil, false
	}
	return c, true
}

// tunnel pipes bytes between client and upstream until both sides hit EOF,
// closes both conns, and returns the (up, down) byte counts.
func tunnel(client, upstream net.Conn) (up, down int64) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(upstream, client)
		atomic.AddInt64(&up, n)
		closeWrite(upstream)
	}()
	go func() {
		defer wg.Done()
		n, _ := io.Copy(client, upstream)
		atomic.AddInt64(&down, n)
		closeWrite(client)
	}()
	wg.Wait()
	_ = upstream.Close()
	_ = client.Close()
	return atomic.LoadInt64(&up), atomic.LoadInt64(&down)
}

func (p *Proxy) record(host, decision string, up, down, durMS int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.rows[host]
	if s == nil {
		s = &hostStat{decision: decision}
		p.rows[host] = s
	}
	// A single deny on a host marks the row as deny; allow rows accumulate.
	if decision == audit.EgressDeny {
		s.decision = audit.EgressDeny
	}
	s.bytesUp += up
	s.bytesDown += down
	s.durationMS += durMS
}

func (p *Proxy) now() time.Time {
	if p.Now == nil {
		return time.Now()
	}
	return p.Now()
}

// closeWrite half-closes the write side of a connection if it supports it.
// Lets the peer see EOF on its read side without tearing down the read side
// of this connection.
func closeWrite(c net.Conn) {
	type cw interface{ CloseWrite() error }
	if w, ok := c.(cw); ok {
		_ = w.CloseWrite()
		return
	}
	// Fall back to a full close if the conn does not split the directions.
	_ = c.Close()
}

// ErrAlreadyStarted is returned if Start is called twice on the same Proxy.
var ErrAlreadyStarted = errors.New("egress: proxy already started")
