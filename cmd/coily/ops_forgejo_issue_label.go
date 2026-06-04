package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoIssueLabelCommand is the `issue label` subtree under
// `coily ops forgejo issue`. Distinct from the top-level `label` subtree,
// which does label-definition CRUD: this assigns/unassigns existing
// labels on a specific issue. Labels are taken by name and resolved to
// ids internally, against both the repo's and the owning org's label sets
// (priority tiers like P0-P4 are org-level labels, invisible to a
// repo-only lookup). Closes coilyco-bridge/coily#159.
func (r *Runner) forgejoIssueLabelCommand() *cli.Command {
	return &cli.Command{
		Name:  "label",
		Usage: "Add, remove, or replace labels on a forgejo issue (by name).",
		Commands: []*cli.Command{
			r.forgejoIssueLabelAddCommand(),
			r.forgejoIssueLabelRemoveCommand(),
			r.forgejoIssueLabelSetCommand(),
		},
	}
}

// forgejoIssueNumberFlag is the shared issue-number flag for the label
// verbs. `--index` is accepted as an alias of `--number` so the spelling
// in coilyco-bridge/coily#159 works and the verbs still match the rest of
// the `issue` subtree, which standardized on `--number`.
func forgejoIssueNumberFlag() *cli.IntFlag {
	return &cli.IntFlag{Name: "number", Aliases: []string{"index"}, Usage: "issue number", Required: true}
}

func forgejoIssueLabelArgs(c *cli.Command) (map[string]string, []string) {
	return map[string]string{
		"--repo":   c.String("repo"),
		"--number": strconv.Itoa(c.Int("number")),
		"--label":  strings.Join(c.StringSlice("label"), ","),
	}, c.Args().Slice()
}

func (r *Runner) forgejoIssueLabelAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add one or more labels to a forgejo issue (repeat --label).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			forgejoIssueNumberFlag(),
			&cli.StringSliceFlag{Name: "label", Usage: "label name to add (repeatable)", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:     "ops.forgejo.issue.label.add",
				ArgsFunc: forgejoIssueLabelArgs,
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoIssueLabelWrite(ctx, c, "add")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueLabelSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Replace the full label set on a forgejo issue (no --label clears all).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			forgejoIssueNumberFlag(),
			&cli.StringSliceFlag{Name: "label", Usage: "label name for the replacement set (repeatable; omit to clear)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:     "ops.forgejo.issue.label.set",
				ArgsFunc: forgejoIssueLabelArgs,
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoIssueLabelWrite(ctx, c, "set")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueLabelRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "Remove one or more labels from a forgejo issue (repeat --label).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			forgejoIssueNumberFlag(),
			&cli.StringSliceFlag{Name: "label", Usage: "label name to remove (repeatable)", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:     "ops.forgejo.issue.label.remove",
				ArgsFunc: forgejoIssueLabelArgs,
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runForgejoIssueLabelRemove(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

// runForgejoIssueLabelWrite handles add (POST) and set (PUT), which share
// the {"labels":[ids]} payload and differ only in method and whether an
// empty set is allowed. Both resolve names to ids first, then go through
// the shared HTTP path so the org-split redirect refusal applies.
func (r *Runner) runForgejoIssueLabelWrite(ctx context.Context, c *cli.Command, mode string) error {
	prefix := "ops forgejo issue label " + mode
	owner, repo, num, names, err := parseForgejoIssueLabelArgs(c, prefix)
	if err != nil {
		return err
	}
	if mode == "add" && len(names) == 0 {
		return fmt.Errorf("%s: at least one --label is required", prefix)
	}
	ids, err := r.forgejoResolveLabelIDs(ctx, prefix, owner, repo, names)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(forgejoIssueLabelsBody{Labels: ids})
	if err != nil {
		return fmt.Errorf("%s: marshal payload: %w", prefix, err)
	}
	method := http.MethodPost
	if mode == "set" {
		method = http.MethodPut
	}
	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels", owner, repo, num)
	respBody, err := r.forgejoAPIDo(ctx, prefix, method, path, payload, http.StatusOK)
	if err != nil {
		return err
	}
	return printForgejoIssueLabelResult(owner, repo, num, respBody)
}

// runForgejoIssueLabelRemove deletes each named label from the issue.
// Forgejo's remove endpoint is one-label-per-call (DELETE .../labels/{id},
// 204), so this resolves every name to an id and issues a DELETE per id.
func (r *Runner) runForgejoIssueLabelRemove(ctx context.Context, c *cli.Command) error {
	prefix := "ops forgejo issue label remove"
	owner, repo, num, names, err := parseForgejoIssueLabelArgs(c, prefix)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("%s: at least one --label is required", prefix)
	}
	ids, err := r.forgejoResolveLabelIDs(ctx, prefix, owner, repo, names)
	if err != nil {
		return err
	}
	for i, id := range ids {
		path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels/%d", owner, repo, num, id)
		if _, derr := r.forgejoAPIDo(ctx, prefix, http.MethodDelete, path, nil, http.StatusNoContent); derr != nil {
			return fmt.Errorf("%s: removing %q: %w", prefix, names[i], derr)
		}
	}
	fmt.Printf("removed %s from %s/%s#%d\n", strings.Join(names, ", "), owner, repo, num)
	return nil
}

// parseForgejoIssueLabelArgs validates the shared --repo / --number /
// --label inputs for every issue-label verb.
func parseForgejoIssueLabelArgs(c *cli.Command, prefix string) (string, string, int, []string, error) {
	if c.Args().Len() != 0 {
		return "", "", 0, nil, fmt.Errorf("%s: takes no positional args, got %d", prefix, c.Args().Len())
	}
	owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
	if err != nil {
		return "", "", 0, nil, err
	}
	num, err := validateForgejoIssueNumber(c.Int("number"), prefix)
	if err != nil {
		return "", "", 0, nil, err
	}
	var names []string
	for _, raw := range c.StringSlice("label") {
		name, verr := validateForgejoLabelName(raw, prefix)
		if verr != nil {
			return "", "", 0, nil, verr
		}
		names = append(names, name)
	}
	return owner, repo, num, names, nil
}

// forgejoResolveLabelIDs maps label names to their numeric ids, checking
// the repo's labels first and then the owning org's labels. Org-level
// labels (priority tiers, icebox) are the common case for cross-repo
// conventions and never show up in a repo-only lookup, so a repo-only
// resolver would spuriously report them missing. Returns a single error
// naming every unresolved label plus what is available.
func (r *Runner) forgejoResolveLabelIDs(ctx context.Context, prefix, owner, repo string, names []string) ([]int, error) {
	nameToID := map[string]int{}

	repoPath := fmt.Sprintf("/api/v1/repos/%s/%s/labels?limit=100", owner, repo)
	repoBody, err := r.forgejoAPIDo(ctx, prefix, http.MethodGet, repoPath, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	mergeForgejoLabels(nameToID, repoBody)

	// Org labels are best-effort: a user-owned repo has no org label set
	// and the endpoint 404s, which is not an error for resolution.
	orgPath := fmt.Sprintf("/api/v1/orgs/%s/labels?limit=100", owner)
	if orgBody, oerr := r.forgejoAPIDo(ctx, prefix, http.MethodGet, orgPath, nil, http.StatusOK); oerr == nil {
		mergeForgejoLabels(nameToID, orgBody)
	}

	return matchForgejoLabelIDs(nameToID, names, prefix)
}

// matchForgejoLabelIDs maps requested label names to ids against a
// resolved name->id table, preserving request order. Split out from the
// HTTP-bound resolver so the match + not-found error is unit-testable.
// On any miss it returns one error naming every unresolved label and the
// sorted set of what is available.
func matchForgejoLabelIDs(nameToID map[string]int, names []string, prefix string) ([]int, error) {
	ids := make([]int, 0, len(names))
	var missing []string
	for _, n := range names {
		if id, ok := nameToID[n]; ok {
			ids = append(ids, id)
		} else {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		avail := make([]string, 0, len(nameToID))
		for k := range nameToID {
			avail = append(avail, k)
		}
		sort.Strings(avail)
		return nil, fmt.Errorf("%s: label(s) not found: %s (available: %s)", prefix, strings.Join(missing, ", "), strings.Join(avail, ", "))
	}
	return ids, nil
}

// mergeForgejoLabels decodes a forgejo label-list response and folds its
// name->id pairs into dst without overwriting entries already present, so
// a repo label shadows an org label of the same name.
func mergeForgejoLabels(dst map[string]int, body []byte) {
	var ls []forgejoLabelSummary
	if err := json.Unmarshal(body, &ls); err != nil {
		return
	}
	for _, l := range ls {
		if _, exists := dst[l.Name]; !exists {
			dst[l.Name] = l.ID
		}
	}
}

// forgejoIssueLabelsBody is the POST/PUT payload for the issue-labels
// endpoint. Forgejo accepts an array of label ids.
type forgejoIssueLabelsBody struct {
	Labels []int `json:"labels"`
}

// printForgejoIssueLabelResult renders the label set the add/set response
// returns, so the caller sees the issue's resulting labels.
func printForgejoIssueLabelResult(owner, repo string, num int, respBody []byte) error {
	var ls []forgejoLabelSummary
	if err := json.Unmarshal(respBody, &ls); err != nil {
		return fmt.Errorf("decode forgejo issue-label response: %w", err)
	}
	names := make([]string, 0, len(ls))
	for _, l := range ls {
		names = append(names, l.Name)
	}
	if len(names) == 0 {
		fmt.Printf("%s/%s#%d now has no labels\n", owner, repo, num)
		return nil
	}
	fmt.Printf("%s/%s#%d labels: %s\n", owner, repo, num, strings.Join(names, ", "))
	return nil
}
