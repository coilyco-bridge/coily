package egress_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/egress"
)

// startProxy starts a fresh proxy, returns its URL plus a cleanup that
// stops it and yields the collected rows.
func startProxy(t *testing.T, allow []string, mode egress.Mode) (proxyURL string, drain func() []audit.EgressRow) {
	t.Helper()
	p := egress.New(allow, mode)
	u, err := p.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop() })
	return u, p.Stop
}

// dialThroughProxy issues a CONNECT to the proxy and returns the resulting
// duplex stream (already past the 200 line).
func dialThroughProxy(t *testing.T, proxyURL, hostport string) (net.Conn, error) {
	t.Helper()
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	c, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return nil, err
	}
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", hostport, hostport)
	if _, err := c.Write([]byte(req)); err != nil {
		_ = c.Close()
		return nil, err
	}
	// Read status line.
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	resp := string(buf[:n])
	if !strings.HasPrefix(resp, "HTTP/1.1 200") {
		_ = c.Close()
		return nil, fmt.Errorf("non-200 from proxy: %q", strings.SplitN(resp, "\r\n", 2)[0])
	}
	return c, nil
}

// newHTTPSServer starts a TLS test server and returns its host:port plus a
// CA cert pool that trusts it. The proxy itself does no TLS, so the client
// (us, in the test) must trust the server cert when negotiating through the
// tunnel.
func newHTTPSServer(t *testing.T, body string) (hostport string, pool *x509.CertPool) {
	t.Helper()
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(s.Close)
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	pool = x509.NewCertPool()
	pool.AddCert(s.Certificate())
	return u.Host, pool
}

func TestProxy_AllowedHostForwards(t *testing.T) {
	hostport, pool := newHTTPSServer(t, "hello-egress")
	host, _, _ := net.SplitHostPort(hostport)

	proxyURL, drain := startProxy(t, []string{host}, egress.ModeEnforce)

	tunnel, err := dialThroughProxy(t, proxyURL, hostport)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	tlsConn := tls.Client(tunnel, &tls.Config{
		ServerName: host,
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("tls handshake: %v", err)
	}
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host)
	if _, err := tlsConn.Write([]byte(req)); err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := io.ReadAll(tlsConn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), "hello-egress") {
		t.Errorf("body = %q, want it to contain hello-egress", string(body))
	}
	_ = tlsConn.Close()

	rows := drain()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	if rows[0].Decision != audit.EgressAllow {
		t.Errorf("decision = %q, want allow", rows[0].Decision)
	}
	if rows[0].Host != host {
		t.Errorf("host = %q, want %q", rows[0].Host, host)
	}
	if rows[0].BytesUp == 0 || rows[0].BytesDown == 0 {
		t.Errorf("bytes up=%d down=%d, want both >0", rows[0].BytesUp, rows[0].BytesDown)
	}
}

func TestProxy_DeniedHostReturns403(t *testing.T) {
	proxyURL, drain := startProxy(t, []string{"only.allowed.example"}, egress.ModeEnforce)

	u, _ := url.Parse(proxyURL)
	c, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer c.Close()
	req := "CONNECT denied.example:443 HTTP/1.1\r\nHost: denied.example:443\r\n\r\n"
	if _, err := c.Write([]byte(req)); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	resp := string(buf[:n])
	if !strings.HasPrefix(resp, "HTTP/1.1 403") {
		t.Errorf("status line = %q, want 403", strings.SplitN(resp, "\r\n", 2)[0])
	}

	rows := drain()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	if rows[0].Decision != audit.EgressDeny {
		t.Errorf("decision = %q, want deny", rows[0].Decision)
	}
	if rows[0].Host != "denied.example" {
		t.Errorf("host = %q, want denied.example", rows[0].Host)
	}
}

func TestProxy_ObserveModeAllowsEverything(t *testing.T) {
	hostport, pool := newHTTPSServer(t, "ok")
	host, _, _ := net.SplitHostPort(hostport)

	// Empty allowlist + ModeObserve must still forward.
	proxyURL, drain := startProxy(t, nil, egress.ModeObserve)

	tunnel, err := dialThroughProxy(t, proxyURL, hostport)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	tlsConn := tls.Client(tunnel, &tls.Config{
		ServerName: host,
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("tls handshake: %v", err)
	}
	_, _ = tlsConn.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"))
	_, _ = io.ReadAll(tlsConn)
	_ = tlsConn.Close()

	rows := drain()
	if len(rows) != 1 || rows[0].Decision != audit.EgressAllow {
		t.Errorf("rows = %+v, want one allow row", rows)
	}
}

func TestProxy_AggregatesByHost(t *testing.T) {
	hostport, pool := newHTTPSServer(t, "x")
	host, _, _ := net.SplitHostPort(hostport)

	proxyURL, drain := startProxy(t, []string{host}, egress.ModeEnforce)

	for i := 0; i < 3; i++ {
		tunnel, err := dialThroughProxy(t, proxyURL, hostport)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		tlsConn := tls.Client(tunnel, &tls.Config{
			ServerName: host,
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConn.Handshake(); err != nil {
			t.Fatalf("tls %d: %v", i, err)
		}
		_, _ = tlsConn.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"))
		_, _ = io.ReadAll(tlsConn)
		_ = tlsConn.Close()
	}
	rows := drain()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 aggregated row: %+v", len(rows), rows)
	}
}

func TestProxy_RejectsNonConnect(t *testing.T) {
	proxyURL, _ := startProxy(t, nil, egress.ModeObserve)
	resp, err := http.Get(proxyURL + "/anything") //nolint:noctx
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}
