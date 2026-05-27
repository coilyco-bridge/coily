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

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoActionsTaskListDefaultLimit caps `actions task list` results when
// the operator doesn't pass --limit. 30 matches forgejo's own page size.
const forgejoActionsTaskListDefaultLimit = 30

// forgejoActionsCommand is the `actions` subtree under `coily ops forgejo`.
// Reads workflow runs (Forgejo calls them "tasks") and pulls their logs.
// Read-only on purpose: investigation, not mutation. See coilysiren/coily#109.
func (r *Runner) forgejoActionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "actions",
		Usage: "Forgejo Actions API verbs (HTTP + in-pod log read).",
		Commands: []*cli.Command{
			{
				Name:  "task",
				Usage: "Forgejo Actions task verbs.",
				Commands: []*cli.Command{
					r.forgejoActionsTaskListCommand(),
					r.forgejoActionsTaskLogsCommand(),
				},
			},
		},
	}
}

func (r *Runner) forgejoActionsTaskListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List recent Forgejo Actions tasks for a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "limit", Usage: "max results to return", Value: forgejoActionsTaskListDefaultLimit},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.actions.task.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--limit": strconv.Itoa(c.Int("limit")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo actions task list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					limit := c.Int("limit")
					if limit <= 0 || limit > 50 {
						return fmt.Errorf("ops forgejo actions task list: --limit must be 1..50, got %d", limit)
					}
					q := url.Values{}
					q.Set("limit", strconv.Itoa(limit))
					path := fmt.Sprintf("/api/v1/repos/%s/%s/actions/tasks?%s", owner, repo, q.Encode())
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo actions task list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoActionsTaskList(respBody)
				},
			},
			r.Audit,
		),
	}
}

// forgejoActionsTaskLogsCommand prints the decoded log for one task. Logs
// live zstd-compressed inside the forgejo pod at a sharded path; we cat
// the file out via kubectl exec and decode with zstd on kai-server.
func (r *Runner) forgejoActionsTaskLogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "Print the decoded log for a Forgejo Actions task.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "id", Usage: "task id (from `actions task list`)", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.actions.task.logs",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo": c.String("repo"),
						"--id":   strconv.Itoa(c.Int("id")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo actions task logs: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					id := c.Int("id")
					if id <= 0 {
						return fmt.Errorf("ops forgejo actions task logs: --id must be positive, got %d", id)
					}
					return r.runForgejoActionsLogs(ctx, owner, repo, id)
				},
			},
			r.Audit,
		),
	}
}

// runForgejoActionsLogs streams the decoded task log to stdout. Forgejo
// stores task logs at /var/lib/gitea/actions_log/<owner>/<repo>/<hex>/<id>.log.zst
// inside the forgejo pod. The hex shard depends on the id; we glob across
// all shards rather than hardcode the sharding scheme, since one task id
// matches exactly one file. Decoding happens on kai-server (zstd binary)
// rather than in-pod (pod has no zstd).
func (r *Runner) runForgejoActionsLogs(ctx context.Context, owner, repo string, id int) error {
	host := r.Cfg.KaiServer.TailscaleHost
	if host == "" {
		return fmt.Errorf("ops forgejo actions task logs: kai_server.tailscale_host must be set in config")
	}
	remote := renderForgejoActionsLogsCmd(owner, repo, id)
	if hostIsLocal(host) {
		return r.Runner.Exec(ctx, "sh", "-c", remote)
	}
	user := r.Cfg.KaiServer.SSHUser
	if user == "" {
		return fmt.Errorf("ops forgejo actions task logs: kai_server.ssh_user must be set in config")
	}
	return r.SSH.Stream(ctx, host, user, remote, os.Stdout, os.Stderr)
}

// renderForgejoActionsLogsCmd builds the remote shell command:
//
//	k3s kubectl -n forgejo exec deploy/forgejo -- sh -c 'cat <glob>' | zstd -d
//
// owner, repo, and id are validated upstream (slug shape + positive int),
// so no shell injection surface here; the glob is the path forgejo
// itself writes to.
func renderForgejoActionsLogsCmd(owner, repo string, id int) string {
	inPod := fmt.Sprintf("cat /var/lib/gitea/actions_log/%s/%s/*/%d.log.zst", owner, repo, id)
	parts := []string{
		"k3s", "kubectl", "-n", "forgejo", "exec", "deploy/forgejo", "--",
		"sh", "-c", posixShellQuote(inPod),
	}
	return strings.Join(parts, " ") + " | zstd -d"
}

// forgejoTaskRow is the subset of one entry in the /actions/tasks
// response that `task list` renders. Field names match forgejo's JSON.
type forgejoTaskRow struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_title"`
	Status      string `json:"status"`
	HeadSHA     string `json:"head_sha"`
	Event       string `json:"event"`
	RunNumber   int    `json:"run_number"`
	URL         string `json:"url"`
	UpdatedAt   string `json:"updated_at"`
}

type forgejoTaskList struct {
	WorkflowRuns []forgejoTaskRow `json:"workflow_runs"`
}

func printForgejoActionsTaskList(respBody []byte) error {
	var out forgejoTaskList
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo actions task list: %w", err)
	}
	if len(out.WorkflowRuns) == 0 {
		fmt.Println("(no tasks)")
		return nil
	}
	for _, t := range out.WorkflowRuns {
		head := t.HeadSHA
		if len(head) > 8 {
			head = head[:8]
		}
		fmt.Printf("%d\t%s\trun#%d\t%s\t%s\t%s\n", t.ID, t.Status, t.RunNumber, head, t.Name, t.DisplayName)
	}
	return nil
}
