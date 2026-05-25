package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
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
			r.forgejoRepoEditCommand(),
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
					return printForgejoRepo(respBody)
				},
			},
			r.Audit,
		),
	}
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
