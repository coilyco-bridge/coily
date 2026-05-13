package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestRewriteGHForREST_IssueCreate covers the headline case from #138 -
// `gh issue create` going through GraphQL is what tripped the secondary
// rate limit. The rewriter routes it through `gh api -X POST` instead.
func TestRewriteGHForREST_IssueCreate(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "title+body",
			in:   []string{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--body", "B"},
			want: []string{"api", "-X", "POST", "repos/coilysiren/coily/issues", "-f", "title=T", "-f", "body=B"},
		},
		{
			name: "title+body inline-=",
			in:   []string{"issue", "create", "--repo=coilysiren/coily", "--title=T", "--body=B"},
			want: []string{"api", "-X", "POST", "repos/coilysiren/coily/issues", "-f", "title=T", "-f", "body=B"},
		},
		{
			name: "short flags",
			in:   []string{"issue", "create", "-R", "coilysiren/coily", "-t", "T", "-b", "B"},
			want: []string{"api", "-X", "POST", "repos/coilysiren/coily/issues", "-f", "title=T", "-f", "body=B"},
		},
		{
			name: "multiple labels and assignees",
			in: []string{"issue", "create", "--repo", "coilysiren/coily",
				"--title", "T", "--body", "B",
				"--label", "bug", "--label", "p1",
				"--assignee", "coilysiren"},
			want: []string{"api", "-X", "POST", "repos/coilysiren/coily/issues",
				"-f", "title=T", "-f", "body=B",
				"-f", "labels[]=bug", "-f", "labels[]=p1",
				"-f", "assignees[]=coilysiren"},
		},
		{
			name: "title only (no body)",
			in:   []string{"issue", "create", "--repo", "coilysiren/coily", "--title", "T"},
			want: []string{"api", "-X", "POST", "repos/coilysiren/coily/issues", "-f", "title=T"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rewriteGHForREST(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("rewriteGHForREST(%v):\n got  %v\n want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRewriteGHForREST_IssueCreateDeclinesOnUntranslatable(t *testing.T) {
	// --project, --milestone, --template all need GraphQL fanout - we
	// pass them through unchanged so gh handles the whole thing.
	cases := [][]string{
		{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--project", "MyProject"},
		{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--milestone", "v1"},
		{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--template", "bug.md"},
	}
	for _, in := range cases {
		got := rewriteGHForREST(in)
		if !reflect.DeepEqual(got, in) {
			t.Errorf("expected fall-through for %v, got %v", in, got)
		}
	}
}

func TestRewriteGHForREST_IssueComment(t *testing.T) {
	in := []string{"issue", "comment", "42", "--repo", "coilysiren/coily", "--body", "hello"}
	want := []string{"api", "-X", "POST", "repos/coilysiren/coily/issues/42/comments", "-f", "body=hello"}
	got := rewriteGHForREST(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRewriteGHForREST_IssueCommentDeclinesEditLast(t *testing.T) {
	in := []string{"issue", "comment", "42", "--repo", "coilysiren/coily", "--edit-last", "--body", "x"}
	got := rewriteGHForREST(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("--edit-last should fall through; got %v", got)
	}
}

func TestRewriteGHForREST_IssueCloseReopen(t *testing.T) {
	// Plain close: PATCH state=closed.
	got := rewriteGHForREST([]string{"issue", "close", "42", "--repo", "coilysiren/coily"})
	want := []string{"api", "-X", "PATCH", "repos/coilysiren/coily/issues/42", "-f", "state=closed"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("issue close: got %v, want %v", got, want)
	}

	// Plain reopen: PATCH state=open.
	got = rewriteGHForREST([]string{"issue", "reopen", "42", "--repo", "coilysiren/coily"})
	want = []string{"api", "-X", "PATCH", "repos/coilysiren/coily/issues/42", "-f", "state=open"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("issue reopen: got %v, want %v", got, want)
	}

	// --comment forces fall-through (needs two API calls).
	in := []string{"issue", "close", "42", "--repo", "coilysiren/coily", "--comment", "done"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, in) {
		t.Errorf("--comment should fall through; got %v", got)
	}
}

func TestRewriteGHForREST_PR(t *testing.T) {
	// PR comments hit the issues comments endpoint (PRs are issues).
	in := []string{"pr", "comment", "9", "--repo", "coilysiren/coily", "--body", "hi"}
	want := []string{"api", "-X", "POST", "repos/coilysiren/coily/issues/9/comments", "-f", "body=hi"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("pr comment: got %v, want %v", got, want)
	}

	in = []string{"pr", "close", "9", "--repo", "coilysiren/coily"}
	want = []string{"api", "-X", "PATCH", "repos/coilysiren/coily/pulls/9", "-f", "state=closed"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("pr close: got %v, want %v", got, want)
	}

	in = []string{"pr", "reopen", "9", "--repo", "coilysiren/coily"}
	want = []string{"api", "-X", "PATCH", "repos/coilysiren/coily/pulls/9", "-f", "state=open"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("pr reopen: got %v, want %v", got, want)
	}
}

func TestRewriteGHForREST_IssueEdit(t *testing.T) {
	// Title+body edit: PATCH.
	in := []string{"issue", "edit", "42", "--repo", "coilysiren/coily", "--title", "T2", "--body", "B2"}
	want := []string{"api", "-X", "PATCH", "repos/coilysiren/coily/issues/42", "-f", "title=T2", "-f", "body=B2"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("issue edit: got %v, want %v", got, want)
	}

	// add-label etc. fall through (need RMW we won't attempt).
	in = []string{"issue", "edit", "42", "--repo", "coilysiren/coily", "--add-label", "bug"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, in) {
		t.Errorf("--add-label should fall through; got %v", got)
	}
}

func TestRewriteGHForREST_PassThroughUntouched(t *testing.T) {
	// Things we deliberately leave to gh - search, repo create, project,
	// release. None should be rewritten.
	cases := [][]string{
		{"issue", "list", "--repo", "coilysiren/coily", "--search", "foo"},
		{"pr", "list", "--repo", "coilysiren/coily"},
		{"repo", "list", "coilysiren"},
		{"repo", "create", "newrepo"},
		{"pr", "create", "--title", "T", "--body", "B"},
		{"project", "item-list", "2", "--owner", "coilysiren"},
		{"search", "issues", "rate limit"},
		{"release", "create", "v1"},
		{"auth", "status"},
		{"api", "user"},
	}
	for _, in := range cases {
		got := rewriteGHForREST(in)
		if !reflect.DeepEqual(got, in) {
			t.Errorf("expected fall-through for %v, got %v", in, got)
		}
	}
}

func TestRewriteGHForREST_BodyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.md")
	if err := os.WriteFile(path, []byte("file body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	in := []string{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--body-file", path}
	got := rewriteGHForREST(in)
	want := []string{"api", "-X", "POST", "repos/coilysiren/coily/issues", "-f", "title=T", "-f", "body=file body\n"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRewriteGHForREST_BodyFileMissingDeclines(t *testing.T) {
	in := []string{"issue", "create", "--repo", "coilysiren/coily", "--title", "T", "--body-file", "/nope/does-not-exist"}
	got := rewriteGHForREST(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("expected fall-through on missing body-file, got %v", got)
	}
}

// TestRewriteGHForREST_IssueView covers the accepted-breaking-change
// rewrite from coilysiren/coily#143. `gh issue view --json` returns a
// gh-synthesized shape; REST returns the full issue object. The rewrite
// is intentional: GraphQL's secondary rate limit dominates the cost of
// downstream callers updating to the REST shape.
func TestRewriteGHForREST_IssueView(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "basic",
			in:   []string{"issue", "view", "42", "--repo", "coilysiren/coily"},
			want: []string{"api", "/repos/coilysiren/coily/issues/42"},
		},
		{
			name: "with json (ignored)",
			in:   []string{"issue", "view", "42", "--repo", "coilysiren/coily", "--json", "number,title,body"},
			want: []string{"api", "/repos/coilysiren/coily/issues/42"},
		},
		{
			name: "short repo flag",
			in:   []string{"issue", "view", "42", "-R", "coilysiren/coily"},
			want: []string{"api", "/repos/coilysiren/coily/issues/42"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rewriteGHForREST(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("rewriteGHForREST(%v):\n got  %v\n want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRewriteGHForREST_PRView(t *testing.T) {
	in := []string{"pr", "view", "7", "--repo", "coilysiren/coily"}
	want := []string{"api", "/repos/coilysiren/coily/pulls/7"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRewriteGHForREST_RepoView(t *testing.T) {
	in := []string{"repo", "view", "coilysiren/coily"}
	want := []string{"api", "/repos/coilysiren/coily"}
	if got := rewriteGHForREST(in); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRewriteGHForREST_ViewDeclines(t *testing.T) {
	// --web means the user wants a browser, not a JSON shape; --comments
	// needs a second call we don't replicate; missing --repo or positional
	// also declines.
	cases := [][]string{
		{"issue", "view", "42", "--repo", "coilysiren/coily", "--web"},
		{"issue", "view", "42", "--repo", "coilysiren/coily", "--comments"},
		{"pr", "view", "7", "--repo", "coilysiren/coily", "--web"},
		{"issue", "view", "--repo", "coilysiren/coily"},
		{"issue", "view", "42"},
		{"repo", "view"},
		{"repo", "view", "--web"},
	}
	for _, in := range cases {
		got := rewriteGHForREST(in)
		if !reflect.DeepEqual(got, in) {
			t.Errorf("expected fall-through for %v, got %v", in, got)
		}
	}
}

func TestRewriteGHForREST_TooShort(t *testing.T) {
	if got := rewriteGHForREST(nil); got != nil {
		t.Errorf("nil argv should pass through unchanged; got %v", got)
	}
	if got := rewriteGHForREST([]string{"auth"}); !reflect.DeepEqual(got, []string{"auth"}) {
		t.Errorf("single-token argv should pass through unchanged; got %v", got)
	}
}
