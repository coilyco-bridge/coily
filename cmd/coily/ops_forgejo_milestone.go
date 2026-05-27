package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoMilestoneCommand is the `milestone` subtree under `coily ops forgejo`.
// Closes coilysiren/coily#117.
func (r *Runner) forgejoMilestoneCommand() *cli.Command {
	return &cli.Command{
		Name:  "milestone",
		Usage: "Forgejo milestone CRUD verbs (HTTP).",
		Commands: []*cli.Command{
			r.forgejoMilestoneCreateCommand(),
			r.forgejoMilestoneListCommand(),
			r.forgejoMilestoneViewCommand(),
			r.forgejoMilestoneEditCommand(),
			r.forgejoMilestoneCloseCommand(),
			r.forgejoMilestoneReopenCommand(),
			r.forgejoMilestoneDeleteCommand(),
		},
	}
}

func (r *Runner) forgejoMilestoneCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo milestone.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "title", Usage: "milestone title", Required: true},
			&cli.StringFlag{Name: "description-file", Usage: "optional path to the milestone description text"},
			&cli.StringFlag{Name: "due-on", Usage: "optional due date as YYYY-MM-DD"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.milestone.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":             c.String("repo"),
						"--title":            c.String("title"),
						"--description-file": c.String("description-file"),
						"--due-on":           c.String("due-on"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoMilestoneCreate(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoMilestoneCreate(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo milestone create: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	title := strings.TrimSpace(c.String("title"))
	if title == "" {
		return fmt.Errorf("ops forgejo milestone create: --title is empty")
	}
	body := forgejoMilestoneCreateBody{Title: title}
	if descPath := c.String("description-file"); descPath != "" {
		desc, rerr := readForgejoBodyFile(descPath, "ops forgejo milestone create")
		if rerr != nil {
			return rerr
		}
		body.Description = desc
	}
	if dueRaw := c.String("due-on"); dueRaw != "" {
		due, derr := normalizeForgejoMilestoneDueOn(dueRaw, "ops forgejo milestone create")
		if derr != nil {
			return derr
		}
		body.DueOn = due
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ops forgejo milestone create: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones", owner, repo)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo milestone create", http.MethodPost, path, payload, http.StatusCreated)
	if err != nil {
		return err
	}
	return printForgejoMilestone(respBody)
}

func (r *Runner) forgejoMilestoneListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo milestones in a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "state", Usage: "open | closed | all", Value: "open"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.milestone.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--state": c.String("state"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo milestone list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					state := c.String("state")
					if state != "open" && state != "closed" && state != "all" {
						return fmt.Errorf("ops forgejo milestone list: --state must be open | closed | all, got %q", state)
					}
					q := url.Values{}
					q.Set("state", state)
					path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones?%s", owner, repo, q.Encode())
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo milestone list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoMilestoneList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoMilestoneViewCommand() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "View a single forgejo milestone.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "milestone id", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.milestone.view",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo milestone view: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoMilestoneNumber(c.Int("number"), "ops forgejo milestone view")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo milestone view", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoMilestoneDetail(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoMilestoneEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit an existing forgejo milestone.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "milestone id", Required: true},
			&cli.StringFlag{Name: "title", Usage: "new milestone title (optional)"},
			&cli.StringFlag{Name: "description-file", Usage: "optional path to new description text"},
			&cli.StringFlag{Name: "due-on", Usage: "optional new due date as YYYY-MM-DD"},
			&cli.StringFlag{Name: "state", Usage: "optional new state: open | closed"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.milestone.edit",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":             c.String("repo"),
						"--number":           strconv.Itoa(c.Int("number")),
						"--title":            c.String("title"),
						"--description-file": c.String("description-file"),
						"--due-on":           c.String("due-on"),
						"--state":            c.String("state"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoMilestoneEdit(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) runForgejoMilestoneEdit(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 0 {
		return fmt.Errorf("ops forgejo milestone edit: takes no positional args, got %d", c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return err
	}
	num, err := validateForgejoMilestoneNumber(c.Int("number"), "ops forgejo milestone edit")
	if err != nil {
		return err
	}
	patch, err := buildForgejoMilestonePatch(c)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("ops forgejo milestone edit: marshal payload: %w", err)
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones/%d", owner, repo, num)
	respBody, err := r.forgejoAPIDo(ctx, "ops forgejo milestone edit", http.MethodPatch, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	return printForgejoMilestone(respBody)
}

// buildForgejoMilestonePatch reads the optional edit flags and assembles
// the PATCH body. Returns an error if none of them are set.
func buildForgejoMilestonePatch(c *cli.Command) (forgejoMilestoneEditBody, error) {
	title := strings.TrimSpace(c.String("title"))
	descPath := c.String("description-file")
	dueRaw := strings.TrimSpace(c.String("due-on"))
	stateRaw := strings.TrimSpace(c.String("state"))
	if title == "" && descPath == "" && dueRaw == "" && stateRaw == "" {
		return forgejoMilestoneEditBody{}, fmt.Errorf("ops forgejo milestone edit: at least one of --title, --description-file, --due-on, --state is required")
	}
	patch := forgejoMilestoneEditBody{}
	if title != "" {
		patch.Title = &title
	}
	if err := applyForgejoMilestonePatchOptionals(&patch, descPath, dueRaw, stateRaw); err != nil {
		return forgejoMilestoneEditBody{}, err
	}
	return patch, nil
}

func applyForgejoMilestonePatchOptionals(patch *forgejoMilestoneEditBody, descPath, dueRaw, stateRaw string) error {
	if descPath != "" {
		desc, err := readForgejoBodyFile(descPath, "ops forgejo milestone edit")
		if err != nil {
			return err
		}
		patch.Description = &desc
	}
	if dueRaw != "" {
		due, err := normalizeForgejoMilestoneDueOn(dueRaw, "ops forgejo milestone edit")
		if err != nil {
			return err
		}
		patch.DueOn = &due
	}
	if stateRaw != "" {
		if stateRaw != "open" && stateRaw != "closed" {
			return fmt.Errorf("ops forgejo milestone edit: --state must be open | closed, got %q", stateRaw)
		}
		patch.State = &stateRaw
	}
	return nil
}

func (r *Runner) forgejoMilestoneCloseCommand() *cli.Command {
	return r.forgejoMilestoneStateCommand("close", "closed")
}

func (r *Runner) forgejoMilestoneReopenCommand() *cli.Command {
	return r.forgejoMilestoneStateCommand("reopen", "open")
}

// forgejoMilestoneStateCommand factors the close/reopen pair. Same pattern
// as forgejoIssueStateCommand.
func (r *Runner) forgejoMilestoneStateCommand(name, state string) *cli.Command {
	verbName := "ops.forgejo.milestone." + name
	errPrefix := "ops forgejo milestone " + name
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Set a forgejo milestone's state to %s.", state),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "milestone id", Required: true},
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
					num, err := validateForgejoMilestoneNumber(c.Int("number"), errPrefix)
					if err != nil {
						return err
					}
					payload, err := json.Marshal(forgejoMilestoneEditBody{State: &state})
					if err != nil {
						return fmt.Errorf("%s: marshal payload: %w", errPrefix, err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, errPrefix, http.MethodPatch, path, payload, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoMilestone(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoMilestoneDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a forgejo milestone by id.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "milestone id", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.milestone.delete",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo milestone delete: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoMilestoneNumber(c.Int("number"), "ops forgejo milestone delete")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/milestones/%d", owner, repo, num)
					if _, err := r.forgejoAPIDo(ctx, "ops forgejo milestone delete", http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
						return err
					}
					fmt.Printf("deleted %s/%s milestone #%d\n", owner, repo, num)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// validateForgejoMilestoneNumber rejects non-positive milestone ids.
func validateForgejoMilestoneNumber(n int, prefix string) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("%s: --number must be >= 1, got %d", prefix, n)
	}
	return n, nil
}

// normalizeForgejoMilestoneDueOn accepts a bare YYYY-MM-DD and returns the
// RFC3339 form forgejo's milestone API expects (end-of-day UTC). The user
// doesn't need to know about RFC3339.
func normalizeForgejoMilestoneDueOn(due, prefix string) (string, error) {
	d := strings.TrimSpace(due)
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return "", fmt.Errorf("%s: --due-on must be YYYY-MM-DD, got %q", prefix, due)
	}
	return t.UTC().Format(time.RFC3339), nil
}

// forgejoMilestoneCreateBody / forgejoMilestoneEditBody mirror the forgejo
// milestone endpoints. Edit fields are pointers so unset fields stay out of
// the marshalled JSON (PATCH "missing key = leave alone" semantics).
type forgejoMilestoneCreateBody struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DueOn       string `json:"due_on,omitempty"`
}

type forgejoMilestoneEditBody struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DueOn       *string `json:"due_on,omitempty"`
	State       *string `json:"state,omitempty"`
}

// forgejoMilestoneSummary is the subset of the forgejo milestone response we
// surface to stdout. Forgejo returns more (counts, timestamps); we ignore
// what's not shown here.
type forgejoMilestoneSummary struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Description string `json:"description"`
	DueOn       string `json:"due_on"`
}

func printForgejoMilestone(respBody []byte) error {
	var out forgejoMilestoneSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo milestone response: %w", err)
	}
	fmt.Printf("#%d\t[%s]\t%s\t%s\n", out.ID, out.State, out.DueOn, out.Title)
	if strings.TrimSpace(out.Description) != "" {
		fmt.Printf("\n%s\n", out.Description)
	}
	return nil
}

func printForgejoMilestoneDetail(respBody []byte) error {
	return printForgejoMilestone(respBody)
}

func printForgejoMilestoneList(respBody []byte) error {
	var out []forgejoMilestoneSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo milestone list: %w", err)
	}
	for _, m := range out {
		fmt.Printf("#%d\t[%s]\t%s\t%s\n", m.ID, m.State, m.DueOn, m.Title)
	}
	return nil
}
