package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"github.com/urfave/cli/v3"
)

// forgejoActionsTaskListDefaultLimit caps `actions task list` results when
// the operator doesn't pass --limit. 30 matches forgejo's own page size.
const forgejoActionsTaskListDefaultLimit = 30

// forgejoActionsCommand is the `actions` subtree under `coily ops forgejo`.
// Reads workflow runs (Forgejo calls them "tasks") and pulls their logs.
// Read-only on purpose: investigation, not mutation. See coilysiren/coily#109.
// Both leaves go over HTTPS - no SSH, no in-pod exec - so they work from a
// session with no SSH key loaded (coilysiren/coily#139).
func (r *Runner) forgejoActionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "actions",
		Usage: "Forgejo Actions API verbs (HTTP).",
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

// forgejoActionsTaskLogsCommand prints the decoded log for one task over
// HTTPS. Forgejo's API v1 has no log endpoint (verified against the live
// swagger on 15.0.2), so this drives the same web route the Actions UI
// uses: POST /{owner}/{repo}/actions/runs/{run}/jobs/{job}/attempt/{n}
// with a logCursors body. Replaces the old SSH + in-pod `cat | zstd -d`
// path so logs are readable without an SSH key (coilysiren/coily#139).
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

// forgejoActionsTaskPageLimit is the per-request page size when scanning
// the tasks list to resolve a task id back to its run + job name. 50 is
// forgejo's API max for this endpoint.
const forgejoActionsTaskPageLimit = 50

// forgejoActionsTaskPageCap bounds how far back the id->run/job scan pages
// before giving up. 50 tasks * 20 pages = the most recent 1000 tasks, which
// comfortably covers any task an operator would still be investigating.
const forgejoActionsTaskPageCap = 20

// runForgejoActionsLogs prints a task's decoded log to stdout over HTTPS.
//
// Three id spaces are in play: the task id the operator passes (shown by
// `task list`), the per-run job index the web log route addresses by, and
// forgejo's internal job id. We bridge them by name: resolve the task id to
// its run number + job name via the tasks API, then match that name against
// the run's job list to get the index. From there the web log route streams
// the per-step log lines.
func (r *Runner) runForgejoActionsLogs(ctx context.Context, owner, repo string, id int) error {
	const prefix = "ops forgejo actions task logs"

	runNumber, jobName, err := r.forgejoResolveTaskJob(ctx, prefix, owner, repo, id)
	if err != nil {
		return err
	}

	// Probe job index 0 to pull the run view (job list + attempt list).
	// Every run has at least one job, and every job has a "Set up job"
	// step, so step 0 of job 0 is a safe cursor for the probe.
	runView, err := r.forgejoFetchJobView(ctx, prefix, owner, repo, runNumber, 0, 1, []int{0})
	if err != nil {
		return err
	}
	jobIndex, err := findForgejoJobIndex(runView.jobNames(), jobName)
	if err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	attempt := runView.maxAttempt()

	// Discover the target job's step count, then request every step's log.
	// Over-requesting steps beyond the job's count makes forgejo 500, so
	// the count has to be exact.
	head, err := r.forgejoFetchJobView(ctx, prefix, owner, repo, runNumber, jobIndex, attempt, []int{0})
	if err != nil {
		return err
	}
	stepCount := len(head.State.CurrentJob.Steps)
	if stepCount == 0 {
		fmt.Printf("(no log steps for task %d)\n", id)
		return nil
	}
	steps := make([]int, stepCount)
	for i := range steps {
		steps[i] = i
	}
	full, err := r.forgejoFetchJobView(ctx, prefix, owner, repo, runNumber, jobIndex, attempt, steps)
	if err != nil {
		return err
	}
	fmt.Print(renderForgejoJobLog(full))
	return nil
}

// forgejoResolveTaskJob maps an Actions task id to its run number + job
// name by paging the tasks list (the id is not directly addressable). The
// id is usually recent, so the first page normally hits; the scan pages
// back up to forgejoActionsTaskPageCap before giving up.
func (r *Runner) forgejoResolveTaskJob(ctx context.Context, prefix, owner, repo string, id int) (int, string, error) {
	for page := 1; page <= forgejoActionsTaskPageCap; page++ {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(forgejoActionsTaskPageLimit))
		q.Set("page", strconv.Itoa(page))
		path := fmt.Sprintf("/api/v1/repos/%s/%s/actions/tasks?%s", owner, repo, q.Encode())
		body, err := r.forgejoAPIDo(ctx, prefix, http.MethodGet, path, nil, http.StatusOK)
		if err != nil {
			return 0, "", err
		}
		var list forgejoTaskList
		if err := json.Unmarshal(body, &list); err != nil {
			return 0, "", fmt.Errorf("%s: decode tasks page %d: %w", prefix, page, err)
		}
		if len(list.WorkflowRuns) == 0 {
			break
		}
		for _, t := range list.WorkflowRuns {
			if t.ID == id {
				if t.RunNumber <= 0 || t.Name == "" {
					return 0, "", fmt.Errorf("%s: task %d has no run number or job name", prefix, id)
				}
				return t.RunNumber, t.Name, nil
			}
		}
	}
	return 0, "", fmt.Errorf("%s: task id %d not found in the most recent %d tasks", prefix, id, forgejoActionsTaskPageLimit*forgejoActionsTaskPageCap)
}

// forgejoFetchJobView POSTs the Actions UI log route for one job attempt and
// decodes the response. steps lists which step cursors to request; pass a
// single-element slice to cheaply probe the run/job/step metadata.
func (r *Runner) forgejoFetchJobView(ctx context.Context, prefix, owner, repo string, runNumber, jobIndex, attempt int, steps []int) (*forgejoJobView, error) {
	payload, err := forgejoLogCursorsBody(steps)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}
	path := fmt.Sprintf("/%s/%s/actions/runs/%d/jobs/%d/attempt/%d", owner, repo, runNumber, jobIndex, attempt)
	body, err := r.forgejoAPIDo(ctx, prefix, http.MethodPost, path, payload, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var view forgejoJobView
	if err := json.Unmarshal(body, &view); err != nil {
		return nil, fmt.Errorf("%s: decode job view: %w", prefix, err)
	}
	return &view, nil
}

// forgejoLogCursorsBody builds the {"logCursors":[...]} POST body the web
// log route expects. expanded=true asks forgejo to return the full step log
// rather than a collapsed tail.
func forgejoLogCursorsBody(steps []int) ([]byte, error) {
	type cursor struct {
		Step     int  `json:"step"`
		Cursor   int  `json:"cursor"`
		Expanded bool `json:"expanded"`
	}
	cursors := make([]cursor, 0, len(steps))
	for _, s := range steps {
		cursors = append(cursors, cursor{Step: s, Cursor: 0, Expanded: true})
	}
	return json.Marshal(struct {
		LogCursors []cursor `json:"logCursors"`
	}{LogCursors: cursors})
}

// findForgejoJobIndex returns the position of jobName within the run's job
// list. Forgejo job names are unique within a run, so name is a stable key
// across the task-id / job-index / job-id id spaces.
func findForgejoJobIndex(jobNames []string, jobName string) (int, error) {
	for i, n := range jobNames {
		if n == jobName {
			return i, nil
		}
	}
	return 0, fmt.Errorf("job %q not found among run jobs %v", jobName, jobNames)
}

// renderForgejoJobLog flattens a job view's per-step log lines into the text
// stream. Steps and lines are sorted by their declared index so output is
// stable regardless of response ordering; each step gets a header so a
// failing step is easy to locate.
func renderForgejoJobLog(view *forgejoJobView) string {
	logs := append([]forgejoStepLog(nil), view.Logs.StepsLog...)
	sort.SliceStable(logs, func(i, j int) bool { return logs[i].Step < logs[j].Step })
	stepNames := view.stepNames()
	var b strings.Builder
	for _, sl := range logs {
		name := ""
		if sl.Step >= 0 && sl.Step < len(stepNames) {
			name = stepNames[sl.Step]
		}
		fmt.Fprintf(&b, "==> [step %d] %s\n", sl.Step, name)
		lines := append([]forgejoLogLine(nil), sl.Lines...)
		sort.SliceStable(lines, func(i, j int) bool { return lines[i].Index < lines[j].Index })
		for _, ln := range lines {
			b.WriteString(ln.Message)
			b.WriteByte('\n')
		}
	}
	return b.String()
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

// forgejoJobView is the subset of the Actions UI job-attempt response that
// log rendering needs. The route returns the whole run context (run.jobs,
// run.allAttempts) alongside the requested step logs.
type forgejoJobView struct {
	State struct {
		Run struct {
			Jobs        []forgejoRunJob     `json:"jobs"`
			AllAttempts []forgejoRunAttempt `json:"allAttempts"`
		} `json:"run"`
		CurrentJob struct {
			Steps []forgejoJobStep `json:"steps"`
		} `json:"currentJob"`
	} `json:"state"`
	Logs struct {
		StepsLog []forgejoStepLog `json:"stepsLog"`
	} `json:"logs"`
}

type forgejoRunJob struct {
	Name string `json:"name"`
}

type forgejoRunAttempt struct {
	Number int `json:"number"`
}

type forgejoJobStep struct {
	Summary string `json:"summary"`
}

type forgejoStepLog struct {
	Step  int              `json:"step"`
	Lines []forgejoLogLine `json:"lines"`
}

type forgejoLogLine struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// jobNames returns the run's job names in index order. The position of a
// name is the job index the web log route addresses by.
func (v *forgejoJobView) jobNames() []string {
	names := make([]string, len(v.State.Run.Jobs))
	for i, j := range v.State.Run.Jobs {
		names[i] = j.Name
	}
	return names
}

// stepNames returns the current job's step summaries in step-index order,
// used to label each step in the rendered log.
func (v *forgejoJobView) stepNames() []string {
	names := make([]string, len(v.State.CurrentJob.Steps))
	for i, s := range v.State.CurrentJob.Steps {
		names[i] = s.Summary
	}
	return names
}

// maxAttempt returns the highest attempt number for the run, defaulting to 1
// when forgejo reports none (the common single-attempt case).
func (v *forgejoJobView) maxAttempt() int {
	highest := 0
	for _, a := range v.State.Run.AllAttempts {
		if a.Number > highest {
			highest = a.Number
		}
	}
	if highest == 0 {
		return 1
	}
	return highest
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
