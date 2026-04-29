package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// auditCommand exposes the audit log path and a tail reader, so an
// external orchestrator (or just the user) can find the file and stream
// records without knowing coily's internal default-path logic.
//
// The path is documented to be:
//  1. $COILY_AUDIT_LOG if set
//  2. audit.log_path from the loaded config layers
//  3. ~/.local/state/coily/audit.jsonl (default)
//
// `coily audit path` always prints whatever pkg/config resolved.
func (r *Runner) auditCommand() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "Inspect the coily audit log.",
		Commands: []*cli.Command{
			r.auditPathCommand(),
			r.auditTailCommand(),
		},
	}
}

func (r *Runner) auditPathCommand() *cli.Command {
	return &cli.Command{
		Name:  "path",
		Usage: "Print the resolved audit log path and exit.",
		Action: verb.Wrap(
			verb.Spec{
				Name:      "audit.path",
				SkipScope: true,
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Println(r.Cfg.Audit.LogPath)
					return nil
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) auditTailCommand() *cli.Command {
	return &cli.Command{
		Name:  "tail",
		Usage: "Stream audit records as JSONL.",
		Description: `tail prints existing records and, with --follow, blocks for new
ones. --since accepts a unix-seconds integer or a relative duration
("5m", "1h", "24h"); records older than that are skipped. Output is
exactly the JSONL lines on disk so a polling consumer can parse them
with jq or any JSON library.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "follow",
				Usage: "block waiting for new records after replaying history",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "skip records older than this (unix seconds or duration like 5m)",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:      "audit.tail",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--since": c.String("since")}, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					since, err := parseSince(c.String("since"))
					if err != nil {
						return err
					}
					return tailAuditLog(ctx, r.Cfg.Audit.LogPath, since, c.Bool("follow"))
				},
			},
			r.Audit,
		),
	}
}

func parseSince(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("audit tail: --since must be unix seconds or a duration: %w", err)
	}
	return time.Now().Add(-d).Unix(), nil
}

// tailAuditLog streams the on-disk JSONL file. Lines older than `since`
// (timestamp filter) are skipped; --follow keeps the file handle open
// and polls for appends every 200ms. Inotify/kqueue would be cleaner
// but the polling latency is fine for the orchestrator-poll case and
// keeps the implementation portable.
func tailAuditLog(ctx context.Context, path string, since int64, follow bool) error {
	f, err := os.Open(path) //nolint:gosec // resolved via pkg/config; reading is the point
	if err != nil {
		return fmt.Errorf("audit tail: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			if matchesSince(line, since) {
				_, _ = io.WriteString(os.Stdout, line)
			}
		}
		switch err {
		case nil:
			continue
		case io.EOF:
			if !follow {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		default:
			return err
		}
	}
}

func matchesSince(line string, since int64) bool {
	if since == 0 {
		return true
	}
	var rec audit.Record
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &rec); err != nil {
		// Malformed line: pass it through rather than swallow silently.
		return true
	}
	return rec.Timestamp >= since
}
