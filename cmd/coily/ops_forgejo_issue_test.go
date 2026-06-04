package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestForgejoRedirectError pins the org-alias silent-no-op fix
// (coilyco-bridge/coily#178, #160): reads may follow a redirect, mutating
// methods must be refused with a canonical --repo suggestion.
func TestForgejoRedirectError(t *testing.T) {
	const alias = "https://f.me/api/v1/repos/alias/coily/issues"
	const canon = "https://f.me/api/v1/repos/canon/coily/issues"

	for _, m := range []string{http.MethodGet, http.MethodHead} {
		if err := forgejoRedirectError(m, alias, canon); err != nil {
			t.Errorf("%s should follow redirects, got %v", m, err)
		}
	}

	for _, m := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		err := forgejoRedirectError(m, alias, canon)
		if err == nil {
			t.Fatalf("%s should refuse the redirect", m)
		}
		if !strings.Contains(err.Error(), "silently no-op") {
			t.Errorf("%s error missing rationale: %v", m, err)
		}
		if !strings.Contains(err.Error(), "--repo canon/coily") {
			t.Errorf("%s error missing canonical --repo suggestion: %v", m, err)
		}
	}
}

// TestForgejoDoWithAliasRetry pins coilyco-bridge/coily#162: a mutating verb
// against a `coilysiren` org-alias owner gets a refused redirect, and the retry
// loop re-issues the same method/body against the canonical owner the redirect
// named rather than bouncing the refusal back to the operator.
func TestForgejoDoWithAliasRetry(t *testing.T) {
	const aliasURL = "https://f.me/api/v1/repos/coilysiren/coily/issues"
	const canonURL = "https://f.me/api/v1/repos/coilyco-bridge/coily/issues"

	t.Run("resolves alias to canonical on a refused mutating redirect", func(t *testing.T) {
		var seen []string
		var notified string
		body, err := forgejoDoWithAliasRetry(aliasURL,
			func(c string) { notified = c },
			func(target string) ([]byte, error) {
				seen = append(seen, target)
				if target == aliasURL {
					return nil, forgejoRedirectError(http.MethodPost, aliasURL, canonURL)
				}
				return []byte("ok"), nil
			})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(body) != "ok" {
			t.Errorf("body = %q, want %q", body, "ok")
		}
		if len(seen) != 2 || seen[0] != aliasURL || seen[1] != canonURL {
			t.Errorf("targets = %v, want [alias canon]", seen)
		}
		if notified != "coilyco-bridge/coily" {
			t.Errorf("notified = %q, want canonical slug", notified)
		}
	})

	t.Run("gives up after maxHops and returns the refusal", func(t *testing.T) {
		calls := 0
		_, err := forgejoDoWithAliasRetry(aliasURL,
			func(string) {},
			func(target string) ([]byte, error) {
				calls++
				return nil, forgejoRedirectError(http.MethodPost, target, canonURL)
			})
		if err == nil {
			t.Fatal("want the redirect refusal after exhausting hops")
		}
		if !strings.Contains(err.Error(), "silently no-op") {
			t.Errorf("error missing rationale: %v", err)
		}
		// One initial call + one retry = 2; the second refusal is returned.
		if calls != forgejoAliasMaxHops+1 {
			t.Errorf("roundTrip calls = %d, want %d", calls, forgejoAliasMaxHops+1)
		}
	})

	t.Run("non-redirect errors are not retried", func(t *testing.T) {
		calls := 0
		_, err := forgejoDoWithAliasRetry(aliasURL,
			func(string) { t.Error("notify should not fire for a plain error") },
			func(string) ([]byte, error) {
				calls++
				return nil, errors.New("boom")
			})
		if err == nil || err.Error() != "boom" {
			t.Errorf("err = %v, want boom passed through", err)
		}
		if calls != 1 {
			t.Errorf("roundTrip calls = %d, want 1 (no retry)", calls)
		}
	})

	t.Run("refusal wrapped through url.Error is still unwrapped", func(t *testing.T) {
		// net/http returns the CheckRedirect error inside a *url.Error, and
		// forgejoAPIRoundTrip wraps that again - errors.As must see through both.
		calls := 0
		body, err := forgejoDoWithAliasRetry(aliasURL,
			func(string) {},
			func(target string) ([]byte, error) {
				calls++
				if target == aliasURL {
					inner := forgejoRedirectError(http.MethodPost, aliasURL, canonURL)
					return nil, fmt.Errorf("ctx: %w", &url.Error{Op: "Post", URL: aliasURL, Err: inner})
				}
				return []byte("ok"), nil
			})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(body) != "ok" || calls != 2 {
			t.Errorf("body=%q calls=%d, want ok/2", body, calls)
		}
	})
}

func TestForgejoRepoFromAPIPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://f.me/api/v1/repos/canon/coily/issues", "canon/coily", true},
		{"https://f.me/api/v1/repos/o/r/releases", "o/r", true},
		{"https://f.me/api/v1/repos/o/r", "o/r", true},
		{"https://f.me/api/v1/repos/o", "", false},
		{"https://f.me/healthz", "", false},
	}
	for _, c := range cases {
		got, ok := forgejoRepoFromAPIPath(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("forgejoRepoFromAPIPath(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseForgejoRepoSlug(t *testing.T) {
	cases := []struct {
		in      string
		owner   string
		repo    string
		wantErr bool
	}{
		{"coilysiren/coily", "coilysiren", "coily", false},
		{"coilysiren/cli-guard", "coilysiren", "cli-guard", false},
		{"a/b.c_d-e", "a", "b.c_d-e", false},
		{"", "", "", true},
		{"single", "", "", true},
		{"a/b/c", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"-owner/repo", "", "", true},
		{"owner/-repo", "", "", true},
		{"owner/re po", "", "", true},
		{"owner/re;po", "", "", true},
	}
	for _, c := range cases {
		o, r, err := parseForgejoRepoSlug(c.in)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("parseForgejoRepoSlug(%q) err=%v, want err=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && (o != c.owner || r != c.repo) {
			t.Errorf("parseForgejoRepoSlug(%q) = (%q,%q), want (%q,%q)", c.in, o, r, c.owner, c.repo)
		}
	}
}

// TestForgejoIssueNumberFlagIndexAlias pins coilyco-bridge/coily#177: every
// issue subverb that takes --number must also accept --index, matching
// forgejo's own API/UI spelling. The flag is built by one shared helper, so
// parsing `--index N` and reading c.Int("number") proves the alias for all of
// them at once.
func TestForgejoIssueNumberFlagIndexAlias(t *testing.T) {
	for _, name := range []string{"number", "index"} {
		var got int
		cmd := &cli.Command{
			Name:  "view",
			Flags: []cli.Flag{forgejoIssueNumberFlag()},
			Action: func(_ context.Context, c *cli.Command) error {
				got = c.Int("number")
				return nil
			},
		}
		if err := cmd.Run(context.Background(), []string{"view", "--" + name, "42"}); err != nil {
			t.Fatalf("--%s parse: %v", name, err)
		}
		if got != 42 {
			t.Errorf("--%s: c.Int(\"number\") = %d, want 42", name, got)
		}
	}
}

// TestResolveForgejoBody covers the inline --body / --body-file exactly-one-of
// contract added in coilyco-bridge/coily#177.
func TestResolveForgejoBody(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "body.md")
	if err := os.WriteFile(filePath, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("write temp body: %v", err)
	}
	emptyPath := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(emptyPath, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("write empty body: %v", err)
	}

	cases := []struct {
		name     string
		body     string
		bodyFile string
		want     string
		wantErr  string
	}{
		{name: "inline only", body: "inline text", want: "inline text"},
		{name: "file only", bodyFile: filePath, want: "from file\n"},
		{name: "both set", body: "x", bodyFile: filePath, wantErr: "mutually exclusive"},
		{name: "neither set", wantErr: "one of --body or --body-file is required"},
		{name: "inline whitespace", body: "   ", wantErr: "--body is empty"},
		{name: "file empty", bodyFile: emptyPath, wantErr: "is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runResolveForgejoBody(t, tc.body, tc.bodyFile)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("body = %q, want %q", got, tc.want)
			}
		})
	}
}

// runResolveForgejoBody drives resolveForgejoBody through a real cli.Command so
// the flag-reading path (c.String) is exercised exactly as in production.
func runResolveForgejoBody(t *testing.T, body, bodyFile string) (string, error) {
	t.Helper()
	var got string
	var resErr error
	cmd := &cli.Command{
		Name: "comment",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body"},
			&cli.StringFlag{Name: "body-file"},
		},
		Action: func(_ context.Context, c *cli.Command) error {
			got, resErr = resolveForgejoBody(c, "test")
			return nil
		},
	}
	argv := []string{"comment"}
	if body != "" {
		argv = append(argv, "--body", body)
	}
	if bodyFile != "" {
		argv = append(argv, "--body-file", bodyFile)
	}
	if err := cmd.Run(context.Background(), argv); err != nil {
		t.Fatalf("cmd.Run: %v", err)
	}
	return got, resErr
}

func TestForgejoIssueCreateBodyShape(t *testing.T) {
	got, err := json.Marshal(forgejoIssueCreateBody{Title: "t", Body: "b"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"title":"t","body":"b"}`
	if string(got) != want {
		t.Errorf("payload = %s, want %s", got, want)
	}
}
