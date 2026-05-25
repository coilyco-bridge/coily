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

// forgejoReleaseListDefaultLimit caps `release list` results when the
// operator doesn't pass --limit. Matches forgejo's own page size.
const forgejoReleaseListDefaultLimit = 30

// forgejoReleaseCommand is the `release` subtree under `coily ops
// forgejo`. CRUD verbs over the forgejo HTTP API. Closes
// coilysiren/coily#72. Asset upload (multipart) is a separate follow-up.
func (r *Runner) forgejoReleaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "release",
		Usage: "Forgejo release CRUD verbs (HTTP).",
		Commands: []*cli.Command{
			r.forgejoReleaseCreateCommand(),
			r.forgejoReleaseListCommand(),
			r.forgejoReleaseViewCommand(),
			r.forgejoReleaseEditCommand(),
			r.forgejoReleaseDeleteCommand(),
		},
	}
}

func (r *Runner) forgejoReleaseCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo release.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "tag", Usage: "git tag name (created if it does not exist)", Required: true},
			&cli.StringFlag{Name: "name", Usage: "release title (defaults to tag if unset)"},
			&cli.StringFlag{Name: "body-file", Usage: "path to the release notes markdown"},
			&cli.StringFlag{Name: "target", Usage: "target commitish (branch or sha) when creating the tag"},
			&cli.BoolFlag{Name: "draft", Usage: "mark as draft"},
			&cli.BoolFlag{Name: "prerelease", Usage: "mark as prerelease"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.release.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":       c.String("repo"),
						"--tag":        c.String("tag"),
						"--name":       c.String("name"),
						"--body-file":  c.String("body-file"),
						"--target":     c.String("target"),
						"--draft":      strconv.FormatBool(c.Bool("draft")),
						"--prerelease": strconv.FormatBool(c.Bool("prerelease")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoReleaseCreate(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoReleaseCreate(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo release create: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	tag, err := validateForgejoTagName(c.String("tag"), "ops forgejo release create")
	if err != nil {
		return err
	}
	body := forgejoReleaseCreateBody{
		TagName:    tag,
		Name:       strings.TrimSpace(c.String("name")),
		Target:     strings.TrimSpace(c.String("target")),
		Draft:      c.Bool("draft"),
		Prerelease: c.Bool("prerelease"),
	}
	if bp := c.String("body-file"); bp != "" {
		b, rerr := readForgejoBodyFile(bp, "ops forgejo release create")
		if rerr != nil {
			return rerr
		}
		body.Body = b
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ops forgejo release create: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/releases", owner, repo)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo release create", http.MethodPost, path, payload, http.StatusCreated)
	if err != nil {
		return err
	}
	return printForgejoRelease(respBody)
}

func (r *Runner) forgejoReleaseListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo releases in a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "limit", Usage: "max results to return", Value: forgejoReleaseListDefaultLimit},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.release.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--limit": strconv.Itoa(c.Int("limit")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo release list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					limit := c.Int("limit")
					if limit <= 0 || limit > 50 {
						return fmt.Errorf("ops forgejo release list: --limit must be 1..50, got %d", limit)
					}
					q := url.Values{}
					q.Set("limit", strconv.Itoa(limit))
					path := fmt.Sprintf("/api/v1/repos/%s/%s/releases?%s", owner, repo, q.Encode())
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo release list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoReleaseList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoReleaseViewCommand() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "View a single forgejo release.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "release id (see `release list`)", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.release.view",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo": c.String("repo"),
						"--id":   strconv.Itoa(c.Int("id")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo release view: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					id := c.Int("id")
					if id <= 0 {
						return fmt.Errorf("ops forgejo release view: --id must be >= 1, got %d", id)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/releases/%d", owner, repo, id)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo release view", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoReleaseDetail(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoReleaseEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit an existing forgejo release.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "release id", Required: true},
			&cli.StringFlag{Name: "name", Usage: "new title (optional)"},
			&cli.StringFlag{Name: "body-file", Usage: "path to new release notes (optional)"},
			&cli.StringFlag{Name: "tag", Usage: "new git tag name (optional)"},
			&cli.BoolFlag{Name: "draft", Usage: "set draft flag (use --no-draft to clear)"},
			&cli.BoolFlag{Name: "prerelease", Usage: "set prerelease flag (use --no-prerelease to clear)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.release.edit",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":       c.String("repo"),
						"--id":         strconv.Itoa(c.Int("id")),
						"--name":       c.String("name"),
						"--body-file":  c.String("body-file"),
						"--tag":        c.String("tag"),
						"--draft":      strconv.FormatBool(c.Bool("draft")),
						"--prerelease": strconv.FormatBool(c.Bool("prerelease")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoReleaseEdit(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoReleaseEdit(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo release edit: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	id := c.Int("id")
	if id <= 0 {
		return fmt.Errorf("ops forgejo release edit: --id must be >= 1, got %d", id)
	}
	patch, err := buildForgejoReleasePatch(c)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("ops forgejo release edit: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/releases/%d", owner, repo, id)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo release edit", http.MethodPatch, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	return printForgejoRelease(respBody)
}

// buildForgejoReleasePatch assembles the PATCH body from whichever
// optional flags the operator set. Boolean flags use c.IsSet so the
// caller can distinguish "leave alone" from "set to false."
func buildForgejoReleasePatch(c *cli.Command) (forgejoReleaseEditBody, error) {
	patch := forgejoReleaseEditBody{}
	anySet := false
	if v := strings.TrimSpace(c.String("name")); v != "" {
		patch.Name = &v
		anySet = true
	}
	if v := strings.TrimSpace(c.String("tag")); v != "" {
		vt, err := validateForgejoTagName(v, "ops forgejo release edit")
		if err != nil {
			return patch, err
		}
		patch.TagName = &vt
		anySet = true
	}
	if bp := c.String("body-file"); bp != "" {
		b, err := readForgejoBodyFile(bp, "ops forgejo release edit")
		if err != nil {
			return patch, err
		}
		patch.Body = &b
		anySet = true
	}
	if c.IsSet("draft") {
		v := c.Bool("draft")
		patch.Draft = &v
		anySet = true
	}
	if c.IsSet("prerelease") {
		v := c.Bool("prerelease")
		patch.Prerelease = &v
		anySet = true
	}
	if !anySet {
		return patch, fmt.Errorf("ops forgejo release edit: at least one of --name, --tag, --body-file, --draft, --prerelease is required")
	}
	return patch, nil
}

func (r *Runner) forgejoReleaseDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a forgejo release by id.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "release id", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.release.delete",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo": c.String("repo"),
						"--id":   strconv.Itoa(c.Int("id")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo release delete: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					id := c.Int("id")
					if id <= 0 {
						return fmt.Errorf("ops forgejo release delete: --id must be >= 1, got %d", id)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/releases/%d", owner, repo, id)
					if _, err := r.forgejoAPIDo(ctx, "ops forgejo release delete", http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
						return err
					}
					fmt.Printf("deleted %s/%s release #%d\n", owner, repo, id)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// validateForgejoTagName applies a sane character set + length to git
// refs. Same defense-in-depth rationale as the slug validator.
func validateForgejoTagName(tag, prefix string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("%s: --tag is empty", prefix)
	}
	if len(tag) > 200 {
		return "", fmt.Errorf("%s: --tag too long (max 200)", prefix)
	}
	if strings.HasPrefix(tag, "-") {
		return "", fmt.Errorf("%s: --tag must not start with '-'", prefix)
	}
	for _, r := range tag {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '/', r == '+':
		default:
			return "", fmt.Errorf("%s: --tag contains invalid character %q", prefix, r)
		}
	}
	return tag, nil
}

type forgejoReleaseCreateBody struct {
	TagName    string `json:"tag_name"`
	Target     string `json:"target_commitish,omitempty"`
	Name       string `json:"name,omitempty"`
	Body       string `json:"body,omitempty"`
	Draft      bool   `json:"draft,omitempty"`
	Prerelease bool   `json:"prerelease,omitempty"`
}

type forgejoReleaseEditBody struct {
	TagName    *string `json:"tag_name,omitempty"`
	Name       *string `json:"name,omitempty"`
	Body       *string `json:"body,omitempty"`
	Draft      *bool   `json:"draft,omitempty"`
	Prerelease *bool   `json:"prerelease,omitempty"`
}

type forgejoReleaseSummary struct {
	ID         int    `json:"id"`
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	HTMLURL    string `json:"html_url"`
}

func printForgejoRelease(respBody []byte) error {
	var out forgejoReleaseSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo release response: %w", err)
	}
	fmt.Printf("%s #%d %s\t%s\t%s\n", out.HTMLURL, out.ID, out.TagName, releaseStateTag(out.Draft, out.Prerelease), out.Name)
	return nil
}

func printForgejoReleaseDetail(respBody []byte) error {
	var out struct {
		forgejoReleaseSummary
		Body string `json:"body"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo release response: %w", err)
	}
	fmt.Printf("%s #%d %s\t%s\t%s\n", out.HTMLURL, out.ID, out.TagName, releaseStateTag(out.Draft, out.Prerelease), out.Name)
	if strings.TrimSpace(out.Body) != "" {
		fmt.Printf("\n%s\n", out.Body)
	}
	return nil
}

func printForgejoReleaseList(respBody []byte) error {
	var out []forgejoReleaseSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo release list: %w", err)
	}
	for _, rel := range out {
		fmt.Printf("%s\t#%d\t%s\t%s\t%s\n", rel.HTMLURL, rel.ID, rel.TagName, releaseStateTag(rel.Draft, rel.Prerelease), rel.Name)
	}
	return nil
}

func releaseStateTag(draft, prerelease bool) string {
	switch {
	case draft:
		return "[draft]"
	case prerelease:
		return "[prerelease]"
	default:
		return "[release]"
	}
}
