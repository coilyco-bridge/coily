package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoLabelCommand is the `label` subtree under `coily ops forgejo`.
// CRUD verbs over the forgejo HTTP API. Closes coilysiren/coily#71.
func (r *Runner) forgejoLabelCommand() *cli.Command {
	return &cli.Command{
		Name:  "label",
		Usage: "Forgejo label CRUD verbs (HTTP).",
		Commands: []*cli.Command{
			r.forgejoLabelCreateCommand(),
			r.forgejoLabelListCommand(),
			r.forgejoLabelEditCommand(),
			r.forgejoLabelDeleteCommand(),
		},
	}
}

func (r *Runner) forgejoLabelCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo label.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "name", Usage: "label name", Required: true},
			&cli.StringFlag{Name: "color", Usage: "6-hex-digit color (with or without leading '#')", Required: true},
			&cli.StringFlag{Name: "description-file", Usage: "optional path to the label description text"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.label.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":             c.String("repo"),
						"--name":             c.String("name"),
						"--color":            c.String("color"),
						"--description-file": c.String("description-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo label create: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					name, err := validateForgejoLabelName(c.String("name"), "ops forgejo label create")
					if err != nil {
						return err
					}
					color, err := normalizeForgejoLabelColor(c.String("color"), "ops forgejo label create")
					if err != nil {
						return err
					}
					body := forgejoLabelCreateBody{Name: name, Color: color}
					if descPath := c.String("description-file"); descPath != "" {
						desc, rerr := readForgejoBodyFile(descPath, "ops forgejo label create")
						if rerr != nil {
							return rerr
						}
						body.Description = desc
					}
					payload, err := json.Marshal(body)
					if err != nil {
						return fmt.Errorf("ops forgejo label create: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/labels", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo label create", http.MethodPost, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoLabel(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoLabelListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo labels in a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.label.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--repo": c.String("repo")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo label list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/labels", owner, repo)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo label list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoLabelList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoLabelEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit a forgejo label by id.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "label id (see `label list`)", Required: true},
			&cli.StringFlag{Name: "name", Usage: "new name (optional)"},
			&cli.StringFlag{Name: "color", Usage: "new color (optional)"},
			&cli.StringFlag{Name: "description-file", Usage: "optional path to new description text"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.label.edit",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":             c.String("repo"),
						"--id":               strconv.Itoa(c.Int("id")),
						"--name":             c.String("name"),
						"--color":            c.String("color"),
						"--description-file": c.String("description-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoLabelEdit(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

// runForgejoLabelEdit is split out so the edit action's complexity stays
// below the gocognit cap. Builds the PATCH payload from whichever
// optional flags were set, sends it, and prints the result.
func (r *Runner) runForgejoLabelEdit(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo label edit: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	id := c.Int("id")
	if id <= 0 {
		return fmt.Errorf("ops forgejo label edit: --id must be >= 1, got %d", id)
	}
	patch, err := buildForgejoLabelPatch(c)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("ops forgejo label edit: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/labels/%d", owner, repo, id)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo label edit", http.MethodPatch, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	return printForgejoLabel(respBody)
}

// buildForgejoLabelPatch reads the optional --name / --color /
// --description-file flags and assembles the PATCH body. Returns an
// error if none of them are set.
func buildForgejoLabelPatch(c *cli.Command) (forgejoLabelEditBody, error) {
	name := strings.TrimSpace(c.String("name"))
	colorRaw := strings.TrimSpace(c.String("color"))
	descPath := c.String("description-file")
	if name == "" && colorRaw == "" && descPath == "" {
		return forgejoLabelEditBody{}, fmt.Errorf("ops forgejo label edit: at least one of --name, --color, --description-file is required")
	}
	patch := forgejoLabelEditBody{}
	if name != "" {
		vn, err := validateForgejoLabelName(name, "ops forgejo label edit")
		if err != nil {
			return forgejoLabelEditBody{}, err
		}
		patch.Name = &vn
	}
	if colorRaw != "" {
		vc, err := normalizeForgejoLabelColor(colorRaw, "ops forgejo label edit")
		if err != nil {
			return forgejoLabelEditBody{}, err
		}
		patch.Color = &vc
	}
	if descPath != "" {
		desc, err := readForgejoBodyFile(descPath, "ops forgejo label edit")
		if err != nil {
			return forgejoLabelEditBody{}, err
		}
		patch.Description = &desc
	}
	return patch, nil
}

func (r *Runner) forgejoLabelDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a forgejo label by id.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "label id", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.label.delete",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo": c.String("repo"),
						"--id":   strconv.Itoa(c.Int("id")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo label delete: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					id := c.Int("id")
					if id <= 0 {
						return fmt.Errorf("ops forgejo label delete: --id must be >= 1, got %d", id)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/labels/%d", owner, repo, id)
					if _, err := r.forgejoAPIDo(ctx, "ops forgejo label delete", http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
						return err
					}
					fmt.Printf("deleted %s/%s label #%d\n", owner, repo, id)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// forgejoLabelCreateBody / forgejoLabelEditBody mirror the forgejo
// label endpoints. Edit fields are pointers so unset fields stay out of
// the marshalled JSON (PATCH "missing key = leave alone" semantics).
type forgejoLabelCreateBody struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

type forgejoLabelEditBody struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type forgejoLabelSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func printForgejoLabel(respBody []byte) error {
	var out forgejoLabelSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo label response: %w", err)
	}
	fmt.Printf("#%d\t%s\t%s\t%s\n", out.ID, out.Color, out.Name, out.Description)
	return nil
}

func printForgejoLabelList(respBody []byte) error {
	var out []forgejoLabelSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo label list: %w", err)
	}
	for _, l := range out {
		fmt.Printf("#%d\t%s\t%s\t%s\n", l.ID, l.Color, l.Name, l.Description)
	}
	return nil
}

// validateForgejoLabelName enforces a sane character set + length. Forgejo
// permits more, but argv-as-flag and shell-injection shapes must be
// rejected. Allows alnum, dash, underscore, dot, slash, space, colon.
func validateForgejoLabelName(name, prefix string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%s: --name is empty", prefix)
	}
	if len(name) > 50 {
		return "", fmt.Errorf("%s: --name too long (max 50)", prefix)
	}
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("%s: --name must not start with '-'", prefix)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '/', r == ' ', r == ':':
		default:
			return "", fmt.Errorf("%s: --name contains invalid character %q", prefix, r)
		}
	}
	return name, nil
}

// normalizeForgejoLabelColor accepts "RRGGBB" or "#RRGGBB" (case-insensitive)
// and returns the "#rrggbb" form forgejo expects.
func normalizeForgejoLabelColor(color, prefix string) (string, error) {
	c := strings.TrimSpace(color)
	c = strings.TrimPrefix(c, "#")
	if len(c) != 6 {
		return "", fmt.Errorf("%s: --color must be 6 hex digits, got %q", prefix, color)
	}
	for _, r := range c {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return "", fmt.Errorf("%s: --color contains non-hex character %q", prefix, r)
		}
	}
	return "#" + strings.ToLower(c), nil
}
