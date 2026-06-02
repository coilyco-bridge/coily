package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

const forgejoPRListDefaultLimit = 30

// forgejoPRCommand is the `pr` subtree under `coily ops forgejo`.
// Closes coilysiren/coily#74. Inline review comments are a follow-up.
func (r *Runner) forgejoPRCommand() *cli.Command {
	return &cli.Command{
		Name:  "pr",
		Usage: "Forgejo pull-request lifecycle verbs (HTTP).",
		Commands: []*cli.Command{
			r.forgejoPRCreateCommand(),
			r.forgejoPRListCommand(),
			r.forgejoPRViewCommand(),
			r.forgejoPRCloseCommand(),
			r.forgejoPRReopenCommand(),
			r.forgejoPRMergeCommand(),
			r.forgejoPRCommentCommand(),
			r.forgejoPRReviewCommand(),
		},
	}
}

func (r *Runner) forgejoPRCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo pull request.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "head", Usage: "source branch (the branch with changes)", Required: true},
			&cli.StringFlag{Name: "base", Usage: "destination branch", Required: true},
			&cli.StringFlag{Name: "title", Usage: "PR title", Required: true},
			&cli.StringFlag{Name: "body-file", Usage: "path to PR description markdown"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "ops.forgejo.pr.create",
				SkipPolicy: true,
				OnComplete: stampPolicySkipped,
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoPRCreate(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoPRCreate(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo pr create: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	head, err := validateForgejoBranchName(c.String("head"), "ops forgejo pr create", "--head")
	if err != nil {
		return err
	}
	base, err := validateForgejoBranchName(c.String("base"), "ops forgejo pr create", "--base")
	if err != nil {
		return err
	}
	title := strings.TrimSpace(c.String("title"))
	if title == "" {
		return fmt.Errorf("ops forgejo pr create: --title is empty")
	}
	body := forgejoPRCreateBody{Head: head, Base: base, Title: title}
	if bp := c.String("body-file"); bp != "" {
		b, rerr := readForgejoBodyFile(bp, "ops forgejo pr create")
		if rerr != nil {
			return rerr
		}
		body.Body = b
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ops forgejo pr create: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner, repo)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo pr create", http.MethodPost, path, payload, http.StatusCreated)
	if err != nil {
		return err
	}
	return printForgejoPR(respBody)
}

func (r *Runner) forgejoPRListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo pull requests in a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "state", Usage: "open | closed | all", Value: "open"},
			&cli.IntFlag{Name: "limit", Usage: "max results to return", Value: forgejoPRListDefaultLimit},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.pr.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--state": c.String("state"),
						"--limit": strconv.Itoa(c.Int("limit")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo pr list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					state := c.String("state")
					if state != "open" && state != "closed" && state != "all" {
						return fmt.Errorf("ops forgejo pr list: --state must be open | closed | all, got %q", state)
					}
					limit := c.Int("limit")
					if limit <= 0 || limit > 50 {
						return fmt.Errorf("ops forgejo pr list: --limit must be 1..50, got %d", limit)
					}
					q := url.Values{}
					q.Set("state", state)
					q.Set("limit", strconv.Itoa(limit))
					path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls?%s", owner, repo, q.Encode())
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo pr list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoPRList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoPRViewCommand() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "View a single forgejo pull request.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "PR number", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.pr.view",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo pr view: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo pr view")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo pr view", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoPRDetail(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoPRCloseCommand() *cli.Command {
	return r.forgejoPRStateCommand("close", "closed")
}

func (r *Runner) forgejoPRReopenCommand() *cli.Command {
	return r.forgejoPRStateCommand("reopen", "open")
}

func (r *Runner) forgejoPRStateCommand(name, state string) *cli.Command {
	verbName := "ops.forgejo.pr." + name
	errPrefix := "ops forgejo pr " + name
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Set a forgejo PR's state to %s.", state),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "PR number", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: verbName,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("%s: takes no positional args, got %d", errPrefix, c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), errPrefix)
					if err != nil {
						return err
					}
					payload, err := json.Marshal(forgejoIssueStateBody{State: state})
					if err != nil {
						return fmt.Errorf("%s: marshal payload: %w", errPrefix, err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, errPrefix, http.MethodPatch, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoPR(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoPRMergeCommand() *cli.Command {
	return &cli.Command{
		Name:  "merge",
		Usage: "Merge a forgejo pull request.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "PR number", Required: true},
			&cli.StringFlag{Name: "method", Usage: "merge | rebase | squash", Value: "merge"},
			&cli.StringFlag{Name: "title", Usage: "optional merge commit title"},
			&cli.StringFlag{Name: "message-file", Usage: "optional path to merge commit body"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.pr.merge",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":         c.String("repo"),
						"--number":       strconv.Itoa(c.Int("number")),
						"--method":       c.String("method"),
						"--title":        c.String("title"),
						"--message-file": c.String("message-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoPRMerge(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoPRMerge(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo pr merge: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo pr merge")
	if err != nil {
		return err
	}
	method := c.String("method")
	if method != "merge" && method != "rebase" && method != "squash" {
		return fmt.Errorf("ops forgejo pr merge: --method must be merge | rebase | squash, got %q", method)
	}
	body := forgejoPRMergeBody{Do: method, MergeTitleField: strings.TrimSpace(c.String("title"))}
	if mp := c.String("message-file"); mp != "" {
		m, rerr := readForgejoBodyFile(mp, "ops forgejo pr merge")
		if rerr != nil {
			return rerr
		}
		body.MergeMessageField = m
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ops forgejo pr merge: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner, repo, num)
	if _, err := r.forgejoAPIDo(ctx, "ops forgejo pr merge", http.MethodPost, path, payload, http.StatusOK); err != nil {
		return err
	}
	fmt.Printf("merged %s/%s#%d via %s\n", owner, repo, num, method)
	return nil
}

func (r *Runner) forgejoPRCommentCommand() *cli.Command {
	return &cli.Command{
		Name:  "comment",
		Usage: "Add a conversation comment to a forgejo PR (uses the issue comments endpoint).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "PR number", Required: true},
			&cli.StringFlag{Name: "body-file", Usage: "path to the comment markdown", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.pr.comment",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":      c.String("repo"),
						"--number":    strconv.Itoa(c.Int("number")),
						"--body-file": c.String("body-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo pr comment: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo pr comment")
					if err != nil {
						return err
					}
					body, err := readForgejoBodyFile(c.String("body-file"), "ops forgejo pr comment")
					if err != nil {
						return err
					}
					payload, err := json.Marshal(forgejoIssueCommentBody{Body: body})
					if err != nil {
						return fmt.Errorf("ops forgejo pr comment: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo pr comment", http.MethodPost, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoCommentSummary(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoPRReviewCommand() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "Submit a review (APPROVE, REQUEST_CHANGES, COMMENT) on a forgejo PR.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "PR number", Required: true},
			&cli.StringFlag{Name: "event", Usage: "APPROVE | REQUEST_CHANGES | COMMENT", Required: true},
			&cli.StringFlag{Name: "body-file", Usage: "optional path to review body markdown"},
			&cli.StringFlag{Name: "comments-file", Usage: "optional path to a JSON array of inline per-line comments"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.pr.review",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":          c.String("repo"),
						"--number":        strconv.Itoa(c.Int("number")),
						"--event":         c.String("event"),
						"--body-file":     c.String("body-file"),
						"--comments-file": c.String("comments-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoPRReview(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoPRReview(ctx context.Context, c *cli.Command) error {
	owner, repo, num, body, err := parseForgejoPRReviewArgs(c)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ops forgejo pr review: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/reviews", owner, repo, num)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo pr review", http.MethodPost, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	var out struct {
		ID      int    `json:"id"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("ops forgejo pr review: decode response: %w", err)
	}
	fmt.Printf("review #%d [%s] %s\n", out.ID, out.State, out.HTMLURL)
	return nil
}

// parseForgejoPRReviewArgs splits the flag parsing + validation out of
// runForgejoPRReview so the action body stays under the cyclop cap.
func parseForgejoPRReviewArgs(c *cli.Command) (string, string, int, forgejoPRReviewBody, error) {
	if c.Args().Len() != 0 {
		return "", "", 0, forgejoPRReviewBody{}, fmt.Errorf("ops forgejo pr review: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return "", "", 0, forgejoPRReviewBody{}, err
	}
	num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo pr review")
	if err != nil {
		return "", "", 0, forgejoPRReviewBody{}, err
	}
	body, err := buildForgejoPRReviewBody(c)
	if err != nil {
		return "", "", 0, forgejoPRReviewBody{}, err
	}
	return owner, repo, num, body, nil
}

// buildForgejoPRReviewBody assembles the review payload from --event,
// --body-file, --comments-file. Split out so parseForgejoPRReviewArgs
// stays under the cyclop cap.
func buildForgejoPRReviewBody(c *cli.Command) (forgejoPRReviewBody, error) {
	event := c.String("event")
	switch event {
	case "APPROVE", "REQUEST_CHANGES", "COMMENT":
	default:
		return forgejoPRReviewBody{}, fmt.Errorf("ops forgejo pr review: --event must be APPROVE | REQUEST_CHANGES | COMMENT, got %q", event)
	}
	body := forgejoPRReviewBody{Event: event}
	if bp := c.String("body-file"); bp != "" {
		b, err := readForgejoBodyFile(bp, "ops forgejo pr review")
		if err != nil {
			return forgejoPRReviewBody{}, err
		}
		body.Body = b
	}
	if cp := c.String("comments-file"); cp != "" {
		comments, err := loadForgejoPRReviewComments(cp)
		if err != nil {
			return forgejoPRReviewBody{}, err
		}
		body.Comments = comments
	}
	if event == "COMMENT" && body.Body == "" && len(body.Comments) == 0 {
		return forgejoPRReviewBody{}, fmt.Errorf("ops forgejo pr review: --event COMMENT requires --body-file or --comments-file")
	}
	return body, nil
}

// loadForgejoPRReviewComments reads a JSON array of inline review
// comments from disk and validates each entry. Forgejo's API silently
// drops malformed entries, so we fail loudly here instead.
func loadForgejoPRReviewComments(path string) ([]forgejoPRReviewCommentBody, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ops forgejo pr review: read --comments-file %s: %w", path, err)
	}
	var out []forgejoPRReviewCommentBody
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("ops forgejo pr review: parse --comments-file %s: %w", path, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ops forgejo pr review: --comments-file %s contained zero comments", path)
	}
	for i, com := range out {
		if err := validateForgejoPRReviewComment(com, i); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func validateForgejoPRReviewComment(com forgejoPRReviewCommentBody, idx int) error {
	if strings.TrimSpace(com.Path) == "" {
		return fmt.Errorf("ops forgejo pr review: --comments-file[%d]: path is empty", idx)
	}
	if strings.TrimSpace(com.Body) == "" {
		return fmt.Errorf("ops forgejo pr review: --comments-file[%d]: body is empty", idx)
	}
	if com.OldPosition < 0 || com.NewPosition < 0 {
		return fmt.Errorf("ops forgejo pr review: --comments-file[%d]: positions must be >= 0", idx)
	}
	hasOld := com.OldPosition > 0
	hasNew := com.NewPosition > 0
	if hasOld == hasNew {
		return fmt.Errorf("ops forgejo pr review: --comments-file[%d]: exactly one of old_position / new_position must be > 0", idx)
	}
	return nil
}

// validateForgejoBranchName applies the same defensive shape as the tag
// validator. Forgejo's git refs allow more, but argv-as-flag and
// shell-injection shapes must be rejected at this layer.
func validateForgejoBranchName(name, prefix, flag string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%s: %s is empty", prefix, flag)
	}
	if len(name) > 250 {
		return "", fmt.Errorf("%s: %s too long", prefix, flag)
	}
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("%s: %s must not start with '-'", prefix, flag)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '/', r == '+':
		default:
			return "", fmt.Errorf("%s: %s contains invalid character %q", prefix, flag, r)
		}
	}
	return name, nil
}

type forgejoPRCreateBody struct {
	Head  string `json:"head"`
	Base  string `json:"base"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

type forgejoPRMergeBody struct {
	Do                string `json:"Do"`
	MergeTitleField   string `json:"MergeTitleField,omitempty"`
	MergeMessageField string `json:"MergeMessageField,omitempty"`
}

type forgejoPRReviewBody struct {
	Event    string                       `json:"event"`
	Body     string                       `json:"body,omitempty"`
	Comments []forgejoPRReviewCommentBody `json:"comments,omitempty"`
}

// forgejoPRReviewCommentBody is a single inline review comment. Exactly
// one of OldPosition / NewPosition must be > 0 (forgejo treats 0 as
// "not on that side"). Path is the tree path; Body is the comment text.
type forgejoPRReviewCommentBody struct {
	Path        string `json:"path"`
	OldPosition int    `json:"old_position"`
	NewPosition int    `json:"new_position"`
	Body        string `json:"body"`
}

type forgejoPRSummary struct {
	Number  int    `json:"number"`
	State   string `json:"state"`
	Title   string `json:"title"`
	Merged  bool   `json:"merged"`
	HTMLURL string `json:"html_url"`
}

func printForgejoPR(respBody []byte) error {
	var out forgejoPRSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo pr response: %w", err)
	}
	fmt.Printf("%s #%d [%s] %s\n", out.HTMLURL, out.Number, prStateTag(out.State, out.Merged), out.Title)
	return nil
}

func printForgejoPRDetail(respBody []byte) error {
	var out struct {
		forgejoPRSummary
		Body string `json:"body"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo pr response: %w", err)
	}
	fmt.Printf("%s #%d [%s] %s\n", out.HTMLURL, out.Number, prStateTag(out.State, out.Merged), out.Title)
	if strings.TrimSpace(out.Body) != "" {
		fmt.Printf("\n%s\n", out.Body)
	}
	return nil
}

func printForgejoPRList(respBody []byte) error {
	var out []forgejoPRSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo pr list: %w", err)
	}
	for _, p := range out {
		fmt.Printf("%s\t#%d\t[%s]\t%s\n", p.HTMLURL, p.Number, prStateTag(p.State, p.Merged), p.Title)
	}
	return nil
}

func prStateTag(state string, merged bool) string {
	if merged {
		return "merged"
	}
	return state
}
