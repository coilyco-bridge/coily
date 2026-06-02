package main

import (
	"os"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/ghcache"
)

// invalidateGHWrite drops the ghcache entries a `method <path>` write
// affects. Called pre-flight from each write-emitting rewriter so the
// next read sees the post-write state. Pre-flight rather than post-
// success because the rewriter only produces argv; the subprocess
// runs later through the passthrough. If the subprocess fails, the
// cache is just empty and the next read refetches - the same
// correctness behavior as before any caching existed.
func invalidateGHWrite(method, path string) {
	ghcache.Invalidate(method, path)
}

// rewriteJQFile expands the coily-side `--jq-file <path>` (or
// `--jq-file=<path>`) shorthand into gh's native `--jq <content>`.
// Pulled out as its own pass so the metachar gate runs on the raw
// argv (where the value is a clean filesystem path) and the file
// content only enters argv after the gate. Per coilysiren/coily#165.
//
// Falls through unchanged on file-read errors so gh reports the
// missing-path error itself instead of coily synthesizing one.
// Falls through if the flag has no value after it (malformed argv).
func rewriteJQFile(argv []string) []string {
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		switch {
		case tok == "--jq-file":
			rest := argv[i+1:]
			if len(rest) == 0 {
				return argv
			}
			// #nosec G304 G602 -- path is operator-supplied via argv;
			// metachar gate already validated it. Bounds checked above.
			content, err := os.ReadFile(rest[0])
			if err != nil {
				return argv
			}
			out = append(out, "--jq", strings.TrimRight(string(content), "\n"))
			i++
		case strings.HasPrefix(tok, "--jq-file="):
			path := strings.TrimPrefix(tok, "--jq-file=")
			// #nosec G304 -- see above.
			content, err := os.ReadFile(path)
			if err != nil {
				return argv
			}
			out = append(out, "--jq", strings.TrimRight(string(content), "\n"))
		default:
			out = append(out, tok)
		}
	}
	return out
}

// rewriteGHForRESTAndJQFile chains the per-invocation passes that
// must run before the gh subprocess sees argv: --max-age extraction
// (coilysiren/coily#267), jq-file expansion (coily#165), and the REST
// rewriter (coily#138). Wired as the gh entry's ArgvRewriter in
// passthroughs.go.
//
// Order matters. --max-age strip runs first so the duration is stashed
// before downstream passes mutate the slice. The jq-file pass runs next
// so the metachar gate has already cleared the file content. The REST
// rewriter runs last on the already-substituted argv.
func rewriteGHForRESTAndJQFile(argv []string) []string {
	stripped, d, ok := resolveGHMaxAge(argv)
	if ok {
		stashGHMaxAge(d)
	}
	return rewriteGHForREST(rewriteJQFile(stripped))
}

// readBodyFile reads the contents of `path` for `--body-file` translation.
// `-` reads from stdin, matching gh's own convention. Returns the body as a
// string with trailing whitespace preserved (issue bodies are sensitive to
// inline whitespace).
func readBodyFile(path string) (string, error) {
	if path == "-" {
		b, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// rewriteGHForREST rewrites argv for `gh ...` so a known set of subcommands
// that gh implements over GraphQL are reshaped into REST calls via `gh api`.
// The goal is to dodge GitHub's GraphQL secondary rate limit on common write
// paths (issue/PR mutations).
//
// Tracking: coilysiren/coily#138.
//
// Coverage today (REST-rewritten):
//
//   - `gh issue create --repo O/R --title T [--body B | --body-file F]
//     [--label L]... [--assignee A]...` - rewritten unless flags we
//     cannot translate (--project / --template / --milestone-by-title)
//     appear, in which case the original argv falls through.
//   - `gh issue comment N --repo O/R [--body B | --body-file F]`
//   - `gh issue close N --repo O/R [--reason completed|not_planned]`
//     (defaults state_reason=completed; --comment still falls through
//     because the GitHub REST equivalent needs two calls)
//   - `gh issue reopen N --repo O/R`
//   - `gh issue edit N --repo O/R [--title T] [--body B | --body-file F]`
//     (no add/remove-label / add/remove-assignee - those need a read-
//     modify-write cycle we leave to gh itself).
//   - `gh pr comment N --repo O/R [--body B | --body-file F]`
//   - `gh pr close N --repo O/R`
//   - `gh pr reopen N --repo O/R`
//   - `gh issue view N --repo O/R` (any --json fields ignored)
//   - `gh pr view N --repo O/R` (any --json fields ignored)
//   - `gh repo view O/R` (any --json fields ignored)
//
// View rewrites carry an accepted breaking change vs `gh ... --json`:
// callers downstream of `coily ops gh issue view --json ...` see REST
// shape, not gh's GraphQL-synthesized shape. Notable diffs:
//
//   - `url` field is absent; use `html_url`.
//   - `state` is lowercase (`open`/`closed`/`merged`), not uppercase.
//   - Nested fields (e.g. `author.login`) are not flattened.
//   - `--json <fields>` is effectively ignored; callers should `--jq`
//     against the full REST object.
//
// Rationale: GraphQL's secondary rate limit trips multiple times per
// Claude session, and reads dominate org-internal gh usage. The cost of
// each affected caller updating to the REST shape is much smaller than
// the cost of recurring rate-limit breakage. Decline rewrite when
// `--web` is present (different output channel), or `--comments` (needs
// a second call), or the issue/pr positional is missing.
//
// Left as GraphQL on purpose (REST is missing the data or the surface is
// too fanout-heavy to translate without bugs):
//
//   - `gh issue list --search ...` / `gh pr list --search ...` - GraphQL
//     Search API, no clean REST equivalent for free-text search.
//   - `gh repo create / edit / fork / list` - many fields live only on
//     GraphQL (templates, repository topics, default-branch protection).
//   - `gh pr create / edit / merge / review` - head/base resolution and
//     merge-method semantics are non-trivial.
//   - `gh project *` - ProjectsV2 has no REST surface at all.
//   - `gh search *` - search is fundamentally a GraphQL API.
//   - `gh release create / edit` - asset uploads and discussion linkage
//     mix REST and GraphQL; gh's own handling is reliable.
//
// Anything not on either list above falls through unchanged, which is the
// safe default (the worst case is "still uses whatever gh chooses" - the
// rewriter never makes the failure mode worse).
func rewriteGHForREST(argv []string) []string {
	// We need at least two tokens to identify a (group, verb) pair like
	// "issue create" or "pr comment". One-token argvs (e.g. just
	// "auth status") aren't rewrite candidates.
	if len(argv) < 2 {
		return argv
	}

	group := argv[0]
	verb := argv[1]
	rest := argv[2:]

	switch group {
	case "issue":
		return dispatchIssueVerb(verb, rest, argv)
	case "pr":
		return dispatchPRVerb(verb, rest, argv)
	case "repo":
		if verb == "view" {
			return rewriteRepoView(rest, argv)
		}
	}
	return argv
}

func dispatchIssueVerb(verb string, rest, argv []string) []string {
	switch verb {
	case "create":
		return rewriteIssueCreate(rest, argv)
	case "comment":
		return rewriteIssueOrPRComment(rest, argv, "issues")
	case "close":
		return rewriteIssueClose(rest, argv)
	case "reopen":
		return rewriteIssueReopen(rest, argv)
	case "edit":
		return rewriteIssueEdit(rest, argv)
	case "view":
		return rewriteIssueOrPRView(rest, argv, "issues")
	}
	return argv
}

func dispatchPRVerb(verb string, rest, argv []string) []string {
	switch verb {
	case "comment":
		// PR comments land on the issues endpoint (PRs are issues).
		return rewriteIssueOrPRComment(rest, argv, "issues")
	case "close":
		return rewritePRClose(rest, argv)
	case "reopen":
		return rewritePRReopen(rest, argv)
	case "view":
		return rewriteIssueOrPRView(rest, argv, "pulls")
	}
	return argv
}

// rewriteIssueOrPRView rewrites `gh issue view N --repo O/R` or
// `gh pr view N --repo O/R` to `gh api /repos/O/R/{issues,pulls}/N`.
// endpointGroup is "issues" or "pulls". Declines on --web (different
// output channel), --comments (needs a second call), or a missing
// number/repo. --json is recognized but ignored: REST returns the full
// object and callers can --jq against it. A `--jq <expr>` (or `-q`)
// from the original argv is forwarded to the rewritten `gh api` call
// so the caller's filter survives the rewrite. coilysiren/coily#278.
func rewriteIssueOrPRView(rest []string, full []string, endpointGroup string) []string {
	p := walkFlags(rest)
	if p.declined {
		return full
	}
	num := ""
	if len(p.positional) > 0 {
		num = p.positional[0]
	}
	repo := p.first("repo")
	if num == "" || repo == "" {
		return full
	}
	out := []string{"api", "/repos/" + repo + "/" + endpointGroup + "/" + num}
	return appendJQ(out, p)
}

// rewriteRepoView rewrites `gh repo view O/R` to `gh api /repos/O/R`.
// Declines on --web or when the positional isn't owner/repo shape (the
// shorthand `gh repo view` with no positional resolves via the local
// git remote, which gh handles for us). A `--jq <expr>` (or `-q`) from
// the original argv is forwarded so the caller's filter survives the
// rewrite. coilysiren/coily#278.
func rewriteRepoView(rest []string, full []string) []string {
	p := walkFlags(rest)
	if p.declined {
		return full
	}
	if len(p.positional) == 0 {
		return full
	}
	repo := p.positional[0]
	if !strings.Contains(repo, "/") {
		return full
	}
	return appendJQ([]string{"api", "/repos/" + repo}, p)
}

// appendJQ forwards a `--jq <expr>` from the original view argv onto the
// rewritten `gh api` argv. `gh api` accepts `--jq` natively so the
// filter applies to the REST response. No-op when the caller did not
// pass `--jq` / `-q`.
func appendJQ(out []string, p parsedArgs) []string {
	if expr := p.first("jq"); expr != "" {
		out = append(out, "--jq", expr)
	}
	return out
}

// rewriteIssueCreate maps `gh issue create --repo O/R --title T [--body B]
// [--label L]... [--assignee A]...` into a `gh api -X POST` call against
// /repos/{O}/{R}/issues. Declines (falls through) when the argv carries
// flags whose semantics don't survive a naive REST translation.
func rewriteIssueCreate(rest []string, full []string) []string {
	repo, title, body, labels, assignees, declined := parseIssueOrPRCreate(rest)
	if declined || repo == "" || title == "" {
		return full
	}
	invalidateGHWrite("POST", "/repos/"+repo+"/issues")
	out := []string{"api", "-X", "POST", "repos/" + repo + "/issues",
		"-f", "title=" + title,
	}
	if body != "" {
		out = append(out, "-f", "body="+body)
	}
	for _, l := range labels {
		out = append(out, "-f", "labels[]="+l)
	}
	for _, a := range assignees {
		out = append(out, "-f", "assignees[]="+a)
	}
	return out
}

func rewriteIssueOrPRComment(rest []string, full []string, endpointGroup string) []string {
	num, repo, body, declined := parseCommentArgs(rest)
	if declined || num == "" || repo == "" || body == "" {
		return full
	}
	invalidateGHWrite("POST", "/repos/"+repo+"/"+endpointGroup+"/"+num+"/comments")
	return []string{
		"api", "-X", "POST",
		"repos/" + repo + "/" + endpointGroup + "/" + num + "/comments",
		"-f", "body=" + body,
	}
}

// rewriteIssueClose handles `gh issue close N --repo O/R [--reason R]`.
// state_reason defaults to "completed" (matching gh's default) and is
// overridden by --reason. Declines on --comment because the GitHub REST
// equivalent needs two calls (POST comment, then PATCH state) and the
// rewriter returns a single argv.
func rewriteIssueClose(rest []string, full []string) []string {
	num, repo, reason, hasExtras := parseIssueCloseArgs(rest)
	if hasExtras || num == "" || repo == "" {
		return full
	}
	if reason == "" {
		reason = "completed"
	}
	invalidateGHWrite("PATCH", "/repos/"+repo+"/issues/"+num)
	return []string{
		"api", "-X", "PATCH",
		"repos/" + repo + "/issues/" + num,
		"-f", "state=closed",
		"-f", "state_reason=" + reason,
	}
}

func rewriteIssueReopen(rest []string, full []string) []string {
	num, repo, hasExtras := parseStateChangeArgs(rest)
	if hasExtras || num == "" || repo == "" {
		return full
	}
	invalidateGHWrite("PATCH", "/repos/"+repo+"/issues/"+num)
	return []string{
		"api", "-X", "PATCH",
		"repos/" + repo + "/issues/" + num,
		"-f", "state=open",
	}
}

func rewritePRClose(rest []string, full []string) []string {
	num, repo, hasExtras := parseStateChangeArgs(rest)
	if hasExtras || num == "" || repo == "" {
		return full
	}
	invalidateGHWrite("PATCH", "/repos/"+repo+"/pulls/"+num)
	return []string{
		"api", "-X", "PATCH",
		"repos/" + repo + "/pulls/" + num,
		"-f", "state=closed",
	}
}

func rewritePRReopen(rest []string, full []string) []string {
	num, repo, hasExtras := parseStateChangeArgs(rest)
	if hasExtras || num == "" || repo == "" {
		return full
	}
	invalidateGHWrite("PATCH", "/repos/"+repo+"/pulls/"+num)
	return []string{
		"api", "-X", "PATCH",
		"repos/" + repo + "/pulls/" + num,
		"-f", "state=open",
	}
}

// rewriteIssueEdit handles `gh issue edit N --repo O/R [--title T] [--body B
// | --body-file F]`. Declines on add/remove-label, add/remove-assignee, or
// add/remove-project because those need a read-modify-write round that gh
// already handles for us; doing it ourselves would mean two API calls and
// race conditions.
func rewriteIssueEdit(rest []string, full []string) []string {
	num, repo, title, body, hasUntranslatable := parseEditArgs(rest)
	if hasUntranslatable || num == "" || repo == "" {
		return full
	}
	if title == "" && body == "" {
		return full
	}
	invalidateGHWrite("PATCH", "/repos/"+repo+"/issues/"+num)
	out := []string{"api", "-X", "PATCH", "repos/" + repo + "/issues/" + num}
	if title != "" {
		out = append(out, "-f", "title="+title)
	}
	if body != "" {
		out = append(out, "-f", "body="+body)
	}
	return out
}

// parsedArgs is the flat key/value-list form `walkFlags` produces. Map keys
// are normalized long flag names without the leading `--`. Values preserve
// argv order so multi-valued flags (--label, --assignee) survive.
type parsedArgs struct {
	values     map[string][]string
	positional []string
	declined   bool
}

// flagAliases maps each accepted short or alternate flag to its canonical
// long name. The walker normalizes every recognized flag to its long form
// so callers read a single key.
var flagAliases = map[string]string{
	"-R": "repo",
	"-t": "title",
	"-b": "body",
	"-F": "body-file",
	"-l": "label",
	"-a": "assignee",
	"-q": "jq",
}

// untranslatableSet names flags whose presence forces us to decline the
// rewrite. Order matches the prose in rewriteGHForREST's package doc.
var untranslatableSet = map[string]bool{
	"project":         true,
	"template":        true,
	"milestone":       true,
	"add-label":       true,
	"remove-label":    true,
	"add-assignee":    true,
	"remove-assignee": true,
	"add-project":     true,
	"remove-project":  true,
	"edit-last":       true,
	"comment":         true,
	"web":             true,
	"comments":        true,
}

// walkFlags scans argv once and folds it into parsedArgs. Long flags
// (`--name value` or `--name=value`) and the short aliases in flagAliases
// are recognized; positional tokens (anything not starting with `-`) land
// in `positional` in argv order. Boolean-style flags without a value (e.g.
// `--edit-last`) still record an empty entry so callers can detect presence.
// Recognizing the untranslatable set short-circuits the walk and marks
// `declined`.
func walkFlags(rest []string) parsedArgs {
	out := parsedArgs{values: map[string][]string{}}
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if tok == "" || !strings.HasPrefix(tok, "-") {
			out.positional = append(out.positional, tok)
			continue
		}
		name, val, hasInline := splitFlag(tok)
		if alias, ok := flagAliases[tok]; ok && !hasInline {
			name = alias
		}
		if untranslatableSet[name] {
			out.declined = true
			return out
		}
		if !hasInline {
			// Consume the next argv if there is one; otherwise this is a
			// bool-style flag and we record presence as "".
			if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "-") {
				val = rest[i+1]
				i++
			}
		}
		out.values[name] = append(out.values[name], val)
	}
	return out
}

// splitFlag normalizes a flag token to (canonical-long-name, value, hasInline).
// Short aliases (`-R`) are mapped via flagAliases; long flags are stripped of
// the leading `--`. Inline-= values stay inline.
func splitFlag(tok string) (name, val string, hasInline bool) {
	if eq := strings.IndexByte(tok, '='); eq >= 0 {
		left := tok[:eq]
		val = tok[eq+1:]
		if alias, ok := flagAliases[left]; ok {
			return alias, val, true
		}
		return strings.TrimPrefix(left, "--"), val, true
	}
	return strings.TrimPrefix(tok, "--"), "", false
}

// first returns the first value collected for a flag, or "".
func (p parsedArgs) first(name string) string {
	if vs := p.values[name]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// parseIssueOrPRCreate scans argv after the `gh issue create` tokens.
// Returns repo, title, body, labels, assignees, and a "declined" bool that's
// true when the argv carries flags we can't faithfully translate.
func parseIssueOrPRCreate(rest []string) (repo, title, body string, labels, assignees []string, declined bool) {
	p := walkFlags(rest)
	if p.declined {
		declined = true
		return
	}
	repo = p.first("repo")
	title = p.first("title")
	body = p.first("body")
	labels = p.values["label"]
	assignees = p.values["assignee"]
	if bodyFile := p.first("body-file"); bodyFile != "" {
		content, err := readBodyFile(bodyFile)
		if err != nil {
			declined = true
			return
		}
		body = content
	}
	return
}

func parseCommentArgs(rest []string) (num, repo, body string, declined bool) {
	p := walkFlags(rest)
	if p.declined {
		declined = true
		return
	}
	if len(p.positional) > 0 {
		num = p.positional[0]
	}
	repo = p.first("repo")
	body = p.first("body")
	if bodyFile := p.first("body-file"); bodyFile != "" {
		content, err := readBodyFile(bodyFile)
		if err != nil {
			declined = true
			return
		}
		body = content
	}
	return
}

// parseIssueCloseArgs is parseStateChangeArgs plus the --reason flag,
// which the GitHub REST PATCH accepts as state_reason. Kept separate so
// reopen / pr close / pr reopen continue to decline on unknown extras.
func parseIssueCloseArgs(rest []string) (num, repo, reason string, hasExtras bool) {
	p := walkFlags(rest)
	if p.declined {
		hasExtras = true
		return
	}
	if len(p.positional) > 0 {
		num = p.positional[0]
	}
	repo = p.first("repo")
	reason = p.first("reason")
	return
}

func parseStateChangeArgs(rest []string) (num, repo string, hasExtras bool) {
	p := walkFlags(rest)
	if p.declined {
		hasExtras = true
		return
	}
	if len(p.positional) > 0 {
		num = p.positional[0]
	}
	repo = p.first("repo")
	return
}

func parseEditArgs(rest []string) (num, repo, title, body string, hasUntranslatable bool) {
	p := walkFlags(rest)
	if p.declined {
		hasUntranslatable = true
		return
	}
	if len(p.positional) > 0 {
		num = p.positional[0]
	}
	repo = p.first("repo")
	title = p.first("title")
	body = p.first("body")
	if bodyFile := p.first("body-file"); bodyFile != "" {
		content, err := readBodyFile(bodyFile)
		if err != nil {
			hasUntranslatable = true
			return
		}
		body = content
	}
	return
}
