// Package ssh is the Go-SDK boundary for ssh and scp. Per
// docs/threat-model.md, simple-API tools (ssh, scp, tailscale) talk to
// remote hosts through Go SDKs instead of shelling out, so no argv ever
// reaches an OS exec layer for these verbs.
//
// This package wraps golang.org/x/crypto/ssh. Authentication uses the
// running ssh-agent when KeyPath is empty, or a private key file when
// set. Host keys are verified against ~/.ssh/known_hosts via
// golang.org/x/crypto/ssh/knownhosts. ssh.InsecureIgnoreHostKey is
// forbidden in this package; if known_hosts cannot be loaded, Dial fails
// closed.
//
// Callers in coily today: cmd/coily/ops_eco.go uses Run to invoke
// systemctl/journalctl on kai-server.
package ssh

import (
	"bytes"
	"context"
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// DefaultDialTimeout is the TCP+SSH-handshake deadline applied when the
// caller's context has no deadline of its own.
const DefaultDialTimeout = 30 * time.Second

// DefaultPort is the standard ssh port. Used when host has no ":port"
// suffix.
const DefaultPort = "22"

// ErrNoAuth is returned when neither ssh-agent nor a private key file
// can supply a usable signer.
var ErrNoAuth = errors.New("ssh: no authentication method available (ssh-agent unreachable and no key path)")

// ErrNoKnownHosts is returned when ~/.ssh/known_hosts cannot be read.
// Failing closed here is the point. ssh.InsecureIgnoreHostKey is never
// used in this package.
var ErrNoKnownHosts = errors.New("ssh: cannot load known_hosts; refusing to dial without host-key verification")

// Client is a configured ssh dialer. Build one per-process and reuse.
// Nil fields take their default values described on each field.
type Client struct {
	// KeyPath is a path to a PEM-encoded private key. If empty, the
	// running ssh-agent (SSH_AUTH_SOCK) is used.
	KeyPath string

	// KnownHostsPath overrides ~/.ssh/known_hosts. Mainly for tests.
	KnownHostsPath string

	// DialTimeout overrides DefaultDialTimeout. Used only when the
	// caller's context has no deadline.
	DialTimeout time.Duration
}

// Run opens a session on host as user, runs cmd, and returns its stdout
// and stderr. The remote command is passed as a single string to the
// remote shell, matching ssh(1) semantics. Callers must construct cmd
// from compile-time constants or values that have passed
// policy.ValidateArg, exactly the same discipline that pkg/shell argv
// requires.
//
// The context bounds the dial and the session. Cancellation closes the
// underlying connection.
func (c *Client) Run(ctx context.Context, host, user, cmd string) (stdout, stderr string, err error) {
	conn, err := c.dial(ctx, host, user)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = conn.Close() }()

	sess, err := conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("ssh: new session: %w", err)
	}
	defer func() { _ = sess.Close() }()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case err := <-done:
		return outBuf.String(), errBuf.String(), err
	case <-ctx.Done():
		// Best-effort cancellation. Closing the session aborts Run.
		_ = sess.Close()
		<-done
		return outBuf.String(), errBuf.String(), ctx.Err()
	}
}

// Stream runs cmd on host as user, streaming stdout/stderr to the
// supplied writers as bytes arrive. Useful for `journalctl -f` style
// verbs where output must be incremental. Returns when the remote
// command exits or ctx is canceled. Either writer may be nil to discard.
func (c *Client) Stream(ctx context.Context, host, user, cmd string, stdout, stderr writer) error {
	conn, err := c.dial(ctx, host, user)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	sess, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("ssh: new session: %w", err)
	}
	defer func() { _ = sess.Close() }()

	if stdout != nil {
		sess.Stdout = stdout
	}
	if stderr != nil {
		sess.Stderr = stderr
	}

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = sess.Close()
		<-done
		return ctx.Err()
	}
}

// writer is the io.Writer subset used by StreamWriters. Exported as an
// unnamed interface so callers can pass *os.File or *bytes.Buffer
// without an extra import dance.
type writer interface {
	Write(p []byte) (int, error)
}

// CopyTo uploads localPath to remotePath on host as user, using SFTP over
// the same ssh transport as Run / Stream. Host-key verification and auth
// go through the same dial() path; ssh.InsecureIgnoreHostKey is still
// forbidden. The remote file is created with mode 0644 and truncated if
// it already exists. Intermediate remote directories are NOT created -
// callers that need them should Run `mkdir -p` first.
//
// remotePath is passed to the remote's SFTP server as-is. Callers must
// build it from compile-time constants or values that have passed
// policy.ValidateArg, same discipline as Run.
func (c *Client) CopyTo(ctx context.Context, host, user, localPath, remotePath string) error {
	conn, err := c.dial(ctx, host, user)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("ssh: sftp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	src, err := os.Open(localPath) // #nosec G304 -- caller-supplied source path is the point of the API
	if err != nil {
		return fmt.Errorf("ssh: open local %s: %w", localPath, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("ssh: create remote %s: %w", remotePath, err)
	}

	done := make(chan error, 1)
	go func() {
		_, cerr := io.Copy(dst, src)
		if closeErr := dst.Close(); cerr == nil {
			cerr = closeErr
		}
		done <- cerr
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ssh: copy to %s: %w", remotePath, err)
		}
		return nil
	case <-ctx.Done():
		_ = dst.Close()
		<-done
		return ctx.Err()
	}
}

// dial resolves auth + host-key verification and opens a connection.
func (c *Client) dial(ctx context.Context, host, user string) (*ssh.Client, error) {
	if user == "" {
		return nil, errors.New("ssh: user is empty")
	}
	if host == "" {
		return nil, errors.New("ssh: host is empty")
	}

	auth, err := c.authMethods()
	if err != nil {
		return nil, err
	}

	hkCallback, err := c.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: hkCallback,
		Timeout:         c.dialTimeout(ctx),
		// Restrict the client's host-key-algorithm list to what is actually
		// recorded in known_hosts for this host. Without this, crypto/ssh
		// requests any server-preferred algo, and when the server offers
		// ecdsa/rsa first we hit "key mismatch" even though an ed25519
		// entry exists. knownhosts.New does not do this automatically in
		// x/crypto v0.50 (no HostKeyDB/HostKeyAlgorithms helper yet).
		HostKeyAlgorithms: c.knownHostAlgos(host),
	}

	addr := withDefaultPort(host)

	// Use a Dialer that respects ctx for the TCP step. crypto/ssh has no
	// context-aware Dial, so we connect manually then hand the conn to
	// ssh.NewClientConn.
	d := net.Dialer{Timeout: cfg.Timeout}
	tcp, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ssh: tcp dial %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(tcp, addr, cfg)
	if err != nil {
		_ = tcp.Close()
		return nil, fmt.Errorf("ssh: handshake %s: %w", addr, err)
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (c *Client) dialTimeout(ctx context.Context) time.Duration {
	if c.DialTimeout > 0 {
		return c.DialTimeout
	}
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 {
			return d
		}
	}
	return DefaultDialTimeout
}

// authMethods returns the auth list. Order: explicit KeyPath first if
// set, else ssh-agent. Returns ErrNoAuth if nothing usable is available.
func (c *Client) authMethods() ([]ssh.AuthMethod, error) {
	if c.KeyPath != "" {
		signer, err := loadSigner(c.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("ssh: load key %s: %w", c.KeyPath, err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, ErrNoAuth
	}
	conn, err := net.Dial("unix", sock) //nolint:gosec // SSH_AUTH_SOCK is the user's own agent
	if err != nil {
		return nil, fmt.Errorf("ssh: dial agent %s: %w", sock, err)
	}
	ag := agent.NewClient(conn)
	signers, err := ag.Signers()
	if err != nil {
		return nil, fmt.Errorf("ssh: agent signers: %w", err)
	}
	if len(signers) == 0 {
		return nil, ErrNoAuth
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signers...)}, nil
}

// hostKeyCallback wires ~/.ssh/known_hosts (or KnownHostsPath if set)
// into the dial. Always strict. ssh.InsecureIgnoreHostKey is forbidden
// in this package and not reachable from any code path.
func (c *Client) hostKeyCallback() (ssh.HostKeyCallback, error) {
	path := c.KnownHostsPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoKnownHosts, err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoKnownHosts, err)
	}
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoKnownHosts, err)
	}
	return cb, nil
}

// knownHostAlgos parses known_hosts and returns the set of host-key
// algorithms recorded for host. Returns nil on any error or empty match
// so crypto/ssh falls back to its default list (which reproduces the
// original "key mismatch" behavior; no worse than today). host may have
// an optional ":port" suffix, which is stripped for the lookup since
// known_hosts bracket-ports only for non-22.
func (c *Client) knownHostAlgos(host string) []string {
	path := c.KnownHostsPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	// Strip :port if present. knownhosts hashes also start with "|1|"; we
	// don't resolve those here, so hashed entries won't contribute. Kai's
	// known_hosts uses plain hostname entries.
	bare := host
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host, "]") {
		bare = host[:i]
	}

	seen := map[string]struct{}{}
	var algos []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// fields[0] is hostlist, fields[1] is algo, fields[2] is base64 key.
		// hostlist may be a comma-separated set. We match exactly; hashed
		// entries (|1|…) are skipped (no hostname match).
		hosts := strings.Split(fields[0], ",")
		match := false
		for _, h := range hosts {
			if h == bare {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		// Sanity-check the base64 blob so a malformed line can't inject
		// an algo name that isn't actually backed by a recorded key.
		if _, err := base64.StdEncoding.DecodeString(fields[2]); err != nil {
			continue
		}
		if _, dup := seen[fields[1]]; dup {
			continue
		}
		seen[fields[1]] = struct{}{}
		algos = append(algos, fields[1])
	}
	return algos
}

func loadSigner(path string) (ssh.Signer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(b)
}

// withDefaultPort returns host as-is if it already has a ":port" suffix,
// otherwise appends DefaultPort. IPv6 literals (which contain colons)
// must be supplied bracketed by the caller.
func withDefaultPort(host string) string {
	if strings.Contains(host, "]") || strings.Count(host, ":") == 1 {
		return host
	}
	return net.JoinHostPort(host, DefaultPort)
}
