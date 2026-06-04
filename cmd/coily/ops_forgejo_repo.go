package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

const forgejoRepoListDefaultLimit = 30

// forgejoRepoCommand is the `repo` subtree under `coily ops forgejo`.
// Closes coilysiren/coily#73.
func (r *Runner) forgejoRepoCommand() *cli.Command {
	return &cli.Command{
		Name:  "repo",
		Usage: "Forgejo repo CRUD verbs (HTTP).",
		Commands: []*cli.Command{
			r.forgejoRepoListCommand(),
			r.forgejoRepoViewCommand(),
			r.forgejoRepoCreateCommand(),
			r.forgejoRepoEditCommand(),
			r.forgejoRepoTopicsCommand(),
			r.forgejoRepoForkCommand(),
			r.forgejoRepoArchiveCommand(),
			r.forgejoRepoDeleteCommand(),
		},
	}
}

func (r *Runner) forgejoRepoListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "Search forgejo repos by name.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Usage: "text to match against repo names"},
			&cli.IntFlag{Name: "limit", Usage: "max results to return", Value: forgejoRepoListDefaultLimit},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--query": c.String("query"),
						"--limit": strconv.Itoa(c.Int("limit")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo list: takes no positional args, got %d", c.Args().Len())
					}
					limit := c.Int("limit")
					if limit <= 0 || limit > 50 {
						return fmt.Errorf("ops forgejo repo list: --limit must be 1..50, got %d", limit)
					}
					q := url.Values{}
					if query := strings.TrimSpace(c.String("query")); query != "" {
						q.Set("q", query)
					}
					q.Set("limit", strconv.Itoa(limit))
					path := "/api/v1/repos/search?" + q.Encode()
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoRepoList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoRepoViewCommand() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "View a single forgejo repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.BoolFlag{Name: "json", Usage: "emit the raw forgejo repo JSON (carries description) instead of the summary line"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.view",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--repo": c.String("repo")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo view: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo view", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					if c.Bool("json") {
						fmt.Println(string(respBody))
						return nil
					}
					return printForgejoRepo(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoRepoCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo repo under an org or the authenticated user (private by default).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "owner", Usage: "destination org or user", Required: true},
			&cli.StringFlag{Name: "name", Usage: "new repo name", Required: true},
			&cli.BoolFlag{Name: "public", Usage: "create the repo public; default is private since public repos are indexed beyond recall, so public must be explicit"},
			&cli.StringFlag{Name: "description", Usage: "repo description"},
			&cli.StringFlag{Name: "default-branch", Usage: "default branch name (e.g. main)"},
			&cli.BoolFlag{Name: "auto-init", Usage: "initialize the repo with a README so the default branch exists"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--owner":          c.String("owner"),
						"--name":           c.String("name"),
						"--public":         strconv.FormatBool(c.Bool("public")),
						"--description":    c.String("description"),
						"--default-branch": c.String("default-branch"),
						"--auto-init":      strconv.FormatBool(c.Bool("auto-init")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoRepoCreate(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoRepoCreate(ctx context.Context, c *cli.Command) error {
	const prefix = "ops forgejo repo create"
	if c.Args().Len() != 0 {
		return fmt.Errorf("%s: takes no positional args, got %d", prefix, c.Args().Len())
	}
	owner := strings.TrimSpace(c.String("owner"))
	name := strings.TrimSpace(c.String("name"))
	if err := validateForgejoCreateOwner(prefix, owner); err != nil {
		return err
	}
	if err := validateForgejoCreateName(prefix, name); err != nil {
		return err
	}
	body := forgejoRepoCreateBodyFrom(name, c.String("description"), c.String("default-branch"), c.Bool("public"), c.Bool("auto-init"))
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("%s: marshal payload: %w", prefix, err)
	}
	// Resolve the token owner so a create aimed at the user's own namespace
	// routes to POST /api/v1/user/repos; any other owner is treated as an org.
	login, err := r.forgejoAuthenticatedLogin(ctx, prefix)
	if err != nil {
		return err
	}
	path := forgejoRepoCreatePath(owner, login)
	respBody, err := r.forgejoAPIDo(ctx, prefix, http.MethodPost, path, payload, http.StatusCreated)
	if err != nil {
		return err
	}
	return printForgejoRepo(respBody)
}

// validateForgejoCreateOwner rejects an empty, leading-hyphen, or
// invalid-character owner. Mirrors parseForgejoRepoSlug's owner checks for the
// create verb, which takes owner and name as separate flags rather than a slug.
func validateForgejoCreateOwner(prefix, owner string) error {
	if owner == "" {
		return fmt.Errorf("%s: --owner is required", prefix)
	}
	if strings.HasPrefix(owner, "-") {
		return fmt.Errorf("%s: --owner %q must not start with '-'", prefix, owner)
	}
	return validateForgejoSlugPart(owner)
}

// validateForgejoCreateName applies the same shape checks to the new repo name.
func validateForgejoCreateName(prefix, name string) error {
	if name == "" {
		return fmt.Errorf("%s: --name is required", prefix)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("%s: --name %q must not start with '-'", prefix, name)
	}
	return validateForgejoSlugPart(name)
}

// forgejoRepoCreateBodyFrom builds the create payload. Private defaults to true
// and is always sent explicitly, so the safe default is recorded on the wire
// rather than left to forgejo's server-side default.
func forgejoRepoCreateBodyFrom(name, description, defaultBranch string, public, autoInit bool) forgejoRepoCreateBody {
	body := forgejoRepoCreateBody{
		Name:    name,
		Private: !public,
	}
	if d := strings.TrimSpace(description); d != "" {
		body.Description = &d
	}
	if b := strings.TrimSpace(defaultBranch); b != "" {
		body.DefaultBranch = &b
	}
	if autoInit {
		v := true
		body.AutoInit = &v
	}
	return body
}

// forgejoRepoCreatePath picks the create endpoint: the authenticated user's own
// namespace routes to POST /api/v1/user/repos, every other owner is treated as
// an org (POST /api/v1/orgs/{org}/repos). Org-alias owners resolve to canonical
// via forgejoAPIDo's redirect retry.
func forgejoRepoCreatePath(owner, login string) string {
	if login != "" && strings.EqualFold(owner, login) {
		return "/api/v1/user/repos"
	}
	return fmt.Sprintf("/api/v1/orgs/%s/repos", owner)
}

// forgejoAuthenticatedLogin returns the login of the token owner via GET
// /api/v1/user, used to decide whether a create targets the user's own
// namespace or an org.
func (r *Runner) forgejoAuthenticatedLogin(ctx context.Context, prefix string) (string, error) {
	respBody, err := r.forgejoAPIDo(ctx, prefix, http.MethodGet, "/api/v1/user", nil, http.StatusOK)
	if err != nil {
		return "", err
	}
	var who struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(respBody, &who); err != nil {
		return "", fmt.Errorf("%s: decode authenticated user: %w", prefix, err)
	}
	if who.Login == "" {
		return "", fmt.Errorf("%s: authenticated user has empty login", prefix)
	}
	return who.Login, nil
}

func (r *Runner) forgejoRepoEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit a forgejo repo's metadata.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "description-file", Usage: "path to new description text"},
			&cli.BoolFlag{Name: "private", Usage: "set private flag"},
			&cli.BoolFlag{Name: "has-issues", Usage: "set has_issues flag"},
			&cli.BoolFlag{Name: "has-wiki", Usage: "set has_wiki flag"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.edit",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":             c.String("repo"),
						"--description-file": c.String("description-file"),
						"--private":          strconv.FormatBool(c.Bool("private")),
						"--has-issues":       strconv.FormatBool(c.Bool("has-issues")),
						"--has-wiki":         strconv.FormatBool(c.Bool("has-wiki")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoRepoEdit(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoRepoEdit(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo repo edit: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	patch, err := buildForgejoRepoPatch(c)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("ops forgejo repo edit: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo edit", http.MethodPatch, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	return printForgejoRepo(respBody)
}

func buildForgejoRepoPatch(c *cli.Command) (forgejoRepoEditBody, error) {
	patch := forgejoRepoEditBody{}
	anySet := false
	if dp := c.String("description-file"); dp != "" {
		d, err := readForgejoBodyFile(dp, "ops forgejo repo edit")
		if err != nil {
			return patch, err
		}
		patch.Description = &d
		anySet = true
	}
	for _, b := range []struct {
		flag string
		dst  **bool
	}{
		{"private", &patch.Private},
		{"has-issues", &patch.HasIssues},
		{"has-wiki", &patch.HasWiki},
	} {
		if c.IsSet(b.flag) {
			v := c.Bool(b.flag)
			*b.dst = &v
			anySet = true
		}
	}
	if !anySet {
		return patch, fmt.Errorf("ops forgejo repo edit: at least one of --description-file, --private, --has-issues, --has-wiki is required")
	}
	return patch, nil
}

func (r *Runner) forgejoRepoForkCommand() *cli.Command {
	return &cli.Command{
		Name:  "fork",
		Usage: "Fork a forgejo repo to the authenticated user or a named org.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "source repo as owner/name", Required: true},
			&cli.StringFlag{Name: "organization", Usage: "destination org (defaults to authenticated user)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.fork",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":         c.String("repo"),
						"--organization": c.String("organization"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo fork: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					body := forgejoRepoForkBody{}
					if org := strings.TrimSpace(c.String("organization")); org != "" {
						if err := validateForgejoSlugPart(org); err != nil {
							return err
						}
						body.Organization = &org
					}
					payload, err := json.Marshal(body)
					if err != nil {
						return fmt.Errorf("ops forgejo repo fork: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/forks", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo fork", http.MethodPost, path, payload, http.StatusAccepted)
					if err != nil {
						return err
					}
					return printForgejoRepo(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoRepoArchiveCommand() *cli.Command {
	return &cli.Command{
		Name:  "archive",
		Usage: "Archive a forgejo repo (sets archived=true).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.archive",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--repo": c.String("repo")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo archive: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					archived := true
					payload, err := json.Marshal(forgejoRepoEditBody{Archived: &archived})
					if err != nil {
						return fmt.Errorf("ops forgejo repo archive: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo archive", http.MethodPatch, path, payload, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoRepo(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoRepoDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a forgejo repo (admin-gated server-side).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.delete",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--repo": c.String("repo")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo delete: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo)
					if _, err := r.forgejoAPIDo(ctx, "ops forgejo repo delete", http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
						return err
					}
					fmt.Printf("deleted %s/%s\n", owner, repo)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// forgejoTopicRe mirrors forgejo's server-side topic constraint: starts with
// an alphanumeric, then alphanumerics or hyphens, up to 35 chars total.
var forgejoTopicRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,34}$`)

func validateForgejoTopic(topic string) error {
	if !forgejoTopicRe.MatchString(topic) {
		return fmt.Errorf("invalid topic %q: must be lowercase alphanumeric or hyphen, start alphanumeric, max 35 chars", topic)
	}
	return nil
}

type forgejoRepoTopicsBody struct {
	Topics []string `json:"topics"`
}

// forgejoRepoTopicsCommand is the `topics` subtree under `coily ops forgejo
// repo`: list (read) and set (replace). Topics are the forgejo equivalent of
// GitHub topics and the source of truth for repo-pointer skill triggers.
func (r *Runner) forgejoRepoTopicsCommand() *cli.Command {
	return &cli.Command{
		Name:  "topics",
		Usage: "List or replace a forgejo repo's topics.",
		Commands: []*cli.Command{
			r.forgejoRepoTopicsListCommand(),
			r.forgejoRepoTopicsSetCommand(),
		},
	}
}

func (r *Runner) forgejoRepoTopicsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List a forgejo repo's topics (one per line).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.topics.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--repo": c.String("repo")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo repo topics list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/topics", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo repo topics list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					var out forgejoRepoTopicsBody
					if err := json.Unmarshal(respBody, &out); err != nil {
						return fmt.Errorf("decode forgejo topics response: %w", err)
					}
					for _, t := range out.Topics {
						fmt.Println(t)
					}
					return nil
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoRepoTopicsSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Replace a forgejo repo's full topic set with --topic values.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringSliceFlag{Name: "topic", Usage: "a topic to set; repeatable. Replaces the existing set."},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.repo.topics.set",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--topic": strings.Join(c.StringSlice("topic"), ","),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoRepoTopicsSet(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoRepoTopicsSet(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo repo topics set: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	topics := c.StringSlice("topic")
	for _, t := range topics {
		if err := validateForgejoTopic(t); err != nil {
			return fmt.Errorf("ops forgejo repo topics set: %w", err)
		}
	}
	// A PUT with an empty list clears all topics. Marshal an explicit [] rather
	// than null so forgejo reads it as "replace with empty set".
	if topics == nil {
		topics = []string{}
	}
	payload, err := json.Marshal(forgejoRepoTopicsBody{Topics: topics})
	if err != nil {
		return fmt.Errorf("ops forgejo repo topics set: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/topics", owner, repo)
	if _, err := r.forgejoAPIDo(ctx, "ops forgejo repo topics set", http.MethodPut, path, payload, http.StatusNoContent); err != nil {
		return err
	}
	fmt.Printf("set %d topic(s) on %s/%s\n", len(topics), owner, repo)
	return nil
}

type forgejoRepoEditBody struct {
	Description *string `json:"description,omitempty"`
	Private     *bool   `json:"private,omitempty"`
	HasIssues   *bool   `json:"has_issues,omitempty"`
	HasWiki     *bool   `json:"has_wiki,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
}

type forgejoRepoForkBody struct {
	Organization *string `json:"organization,omitempty"`
}

type forgejoRepoCreateBody struct {
	Name          string  `json:"name"`
	Private       bool    `json:"private"`
	Description   *string `json:"description,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
	AutoInit      *bool   `json:"auto_init,omitempty"`
}

type forgejoRepoSummary struct {
	ID          int    `json:"id"`
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	Archived    bool   `json:"archived"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
}

func printForgejoRepo(respBody []byte) error {
	var out forgejoRepoSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo repo response: %w", err)
	}
	fmt.Printf("%s\t%s\t%s\t%s\n", out.HTMLURL, out.FullName, repoStateTag(out.Private, out.Archived), out.Description)
	return nil
}

func printForgejoRepoList(respBody []byte) error {
	var out struct {
		Data []forgejoRepoSummary `json:"data"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo repo search: %w", err)
	}
	for _, r := range out.Data {
		fmt.Printf("%s\t%s\t%s\t%s\n", r.HTMLURL, r.FullName, repoStateTag(r.Private, r.Archived), r.Description)
	}
	return nil
}

func repoStateTag(private, archived bool) string {
	switch {
	case archived:
		return "[archived]"
	case private:
		return "[private]"
	default:
		return "[public]"
	}
}
