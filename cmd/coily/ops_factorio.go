package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// factorioCommand wraps the factorio dedicated server which runs as a
// systemd unit on kai-server. The five lifecycle verbs reuse the
// systemdUnit pattern (parity with eco/core-keeper/icarus). The
// factorio-specific additions are:
//
//   - update:  re-run steamcmd against app 427520 (the existing
//     factorio-server-pre.sh) so a stopped server picks up the
//     latest stable headless build before next start.
//   - saves:   list / backup-now against the saves dir on kai-server.
//   - mods:    list mods/ on the server, or sync the mod files on
//     disk to match mod-list.json by pulling missing /
//     out-of-date archives from the Factorio mod portal.
//     The sync itself runs in factorio-mods-sync.sh on
//     kai-server (parity with update / backup-now); coily
//     just provides the audit-logged trigger and forwards
//     --dry-run / --mod for selective runs.
//   - players: list whitelist + ban entries.
//
// Every action is a single ssh-streamed command. No subprocess fork.
func (r *Runner) factorioCommand() *cli.Command {
	unit := systemdUnit{
		VerbName:     "factorio",
		UnitName:     "factorio-server",
		StartEnables: true,
		StopDisables: true,
	}
	return &cli.Command{
		Name:  unit.VerbName,
		Usage: "Operate the factorio-server systemd unit on kai-server.",
		Description: `factorio wraps systemctl/journalctl calls against the factorio-server
unit on kai-server, plus a small set of factorio-specific helpers
(update via steamcmd, saves listing/backup, mods listing, players
listing). The five lifecycle verbs (status/tail/restart/stop/start)
mirror the eco / core-keeper / icarus pattern.`,
		Commands: []*cli.Command{
			r.systemdStatus(unit),
			r.systemdTail(unit),
			r.systemdRestart(unit),
			r.systemdStop(unit),
			r.systemdStart(unit),
			r.factorioUpdateCommand(),
			r.factorioSavesCommand(),
			r.factorioModsCommand(),
			r.factorioPlayersCommand(),
		},
	}
}

// factorioServerDir returns the resolved server install dir on
// kai-server. Falls back to the well-known Steam path when the embedded
// config left it blank, which matches the existing
// factorio-server-{pre,start}.sh scripts.
func (r *Runner) factorioServerDir() string {
	if v := r.Cfg.Factorio.ServerDir; v != "" {
		return v
	}
	return "/home/kai/Steam/steamapps/common/FactorioServer"
}

// factorioRemote runs a single shell command on kai-server and streams
// stdout/stderr back. cmd is composed from compile-time string literals
// plus a small set of validated flags - never raw user input.
func (r *Runner) factorioRemote(cmd string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		host := r.Cfg.KaiServer.TailscaleHost
		user := r.Cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("factorio: kai_server.tailscale_host or ssh_user not configured")
		}
		if err := r.SSH.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("factorio: remote exec: %w", err)
		}
		return nil
	}
}

// factorioUpdateCommand re-runs the existing pre-start steamcmd update
// (app id 427520) so a stopped server picks up the latest stable
// headless build. Equivalent of running factorio-server-pre.sh by hand.
func (r *Runner) factorioUpdateCommand() *cli.Command {
	script := "/home/kai/projects/infrastructure/scripts/factorio-server-pre.sh"
	return &cli.Command{
		Name:  "update",
		Usage: "Run steamcmd against app 427520 to update the factorio install.",
		Description: `update re-runs factorio-server-pre.sh on kai-server. The script
calls steamcmd to validate / pull the latest stable headless binary.
Run this with the server stopped; running it while the unit is active
is harmless but wastes the Steam download.`,
		Action: verb.Wrap(
			verb.Spec{
				Name:   "factorio.update",
				Action: r.factorioRemote("bash " + script),
			},
			r.Audit,
		),
	}
}

func (r *Runner) factorioSavesCommand() *cli.Command {
	return &cli.Command{
		Name:  "saves",
		Usage: "Inspect and back up factorio save files on kai-server.",
		Commands: []*cli.Command{
			r.factorioSavesListCommand(),
			r.factorioSavesBackupCommand(),
		},
	}
}

// factorioSavesListCommand prints the saves directory listing in
// time-sorted order. Read-only; no sudo.
func (r *Runner) factorioSavesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List zip saves under the FactorioServer/saves directory.",
		Action: verb.Wrap(
			verb.Spec{
				Name: "factorio.saves.list",
				Action: r.factorioRemote(fmt.Sprintf(
					"ls -lh --time=mtime %s/saves/*.zip 2>/dev/null || echo '(no saves)'",
					r.factorioServerDir(),
				)),
			},
			r.Audit,
		),
	}
}

// factorioSavesBackupCommand triggers the kai-server-side
// factorio-backup.sh runner. The runner copies the saves dir to
// s3://kai-game-backups/factorio/<host>/<utc-timestamp>/ via the IAM
// creds at /home/kai/.aws/. Idempotent and crash-safe; failure shows
// up in the audit log + the backup script's stderr.
func (r *Runner) factorioSavesBackupCommand() *cli.Command {
	script := "/home/kai/projects/infrastructure/scripts/factorio-backup.sh"
	return &cli.Command{
		Name:  "backup-now",
		Usage: "Trigger an immediate off-cluster snapshot of the saves dir.",
		Description: `backup-now invokes factorio-backup.sh on kai-server, which copies
the FactorioServer/saves directory to the configured S3 bucket
(s3://kai-game-backups/factorio/...). Cron runs the same script
nightly; this verb is for ad-hoc snapshots before risky operations.`,
		Action: verb.Wrap(
			verb.Spec{
				Name:   "factorio.saves.backup-now",
				Action: r.factorioRemote("bash " + script),
			},
			r.Audit,
		),
	}
}

func (r *Runner) factorioModsCommand() *cli.Command {
	return &cli.Command{
		Name:  "mods",
		Usage: "Inspect or sync the mod stack installed on the factorio server.",
		Commands: []*cli.Command{
			r.factorioModsListCommand(),
			r.factorioModsSyncCommand(),
		},
	}
}

// factorioModsListCommand reads mod-list.json on kai-server and prints
// each enabled / disabled entry. mod-list.json is the canonical source
// of truth for what the server loads.
func (r *Runner) factorioModsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "Print mod-list.json entries with their enabled flag.",
		Action: verb.Wrap(
			verb.Spec{
				Name: "factorio.mods.list",
				Action: r.factorioRemote(fmt.Sprintf(
					`if [ -f %s/mods/mod-list.json ]; then `+
						`jq -r '.mods[] | "\(.enabled)\t\(.name)"' %s/mods/mod-list.json | column -t; `+
						`else echo '(no mod-list.json yet)'; fi`,
					r.factorioServerDir(), r.factorioServerDir(),
				)),
			},
			r.Audit,
		),
	}
}

// factorioModsSyncCommand triggers factorio-mods-sync.sh on kai-server.
// The script reads mods/mod-list.json, queries the Factorio mod portal
// for the latest release of each enabled non-base mod that matches the
// installed factorio version, and writes the resulting <name>_<ver>.zip
// archives into mods/. Auth (username + token) is read from the server's
// player-data.json by the script - it never crosses the wire.
//
// --dry-run forwards through so the script can preview adds / updates /
// removals without touching mods/. --mod scopes the sync to a single
// mod name (still validated against mod-list.json server-side). Both
// flags are validated by policy.ValidateArg before they reach the
// remote shell.
func (r *Runner) factorioModsSyncCommand() *cli.Command {
	const script = "/home/kai/projects/infrastructure/scripts/factorio-mods-sync.sh"
	return &cli.Command{
		Name:  "sync",
		Usage: "Pull the mod files in mods/ into agreement with mod-list.json.",
		Description: `sync invokes factorio-mods-sync.sh on kai-server. The script reads
mods/mod-list.json, fetches each enabled mod's latest release that
matches the installed factorio major.minor from the Factorio mod
portal API, and writes the archives into mods/. The base mod is
always skipped (it ships with the dedicated server binary).

The script is the source of truth for the sync algorithm; coily just
provides the audit-logged trigger and forwards two flags:

  --dry-run        preview the planned add / update / remove set
                   without writing to mods/.
  --mod <name>     scope the sync to a single mod (still validated
                   against mod-list.json server-side).

This verb does NOT restart the server. Run 'coily gaming factorio
restart' afterwards once you are happy with the new mod set.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "dry-run", Usage: "preview without writing to mods/"},
			&cli.StringFlag{Name: "mod", Usage: "scope sync to a single mod name"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "factorio.mods.sync",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--dry-run": fmt.Sprint(c.Bool("dry-run")),
						"--mod":     c.String("mod"),
					}, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cmd := "bash " + script
					if c.Bool("dry-run") {
						cmd += " --dry-run"
					}
					if mod := c.String("mod"); mod != "" {
						// policy.ValidateArg has already run via verb.Wrap, but
						// the value is interpolated into a remote shell command,
						// so re-check the literal before sending it over.
						if err := policy.ValidateArg("--mod", mod); err != nil {
							return fmt.Errorf("factorio mods sync: %w", err)
						}
						cmd += " --mod " + mod
					}
					return r.factorioRemote(cmd)(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) factorioPlayersCommand() *cli.Command {
	return &cli.Command{
		Name:  "players",
		Usage: "Inspect the factorio whitelist / banlist / adminlist files.",
		Commands: []*cli.Command{
			r.factorioPlayersListCommand("whitelist", "server-whitelist.json"),
			r.factorioPlayersListCommand("banlist", "server-banlist.json"),
			r.factorioPlayersListCommand("adminlist", "server-adminlist.json"),
		},
	}
}

// factorioPlayersListCommand pretty-prints one of the JSON player
// lists. listFile must be a known filename literal so it can never
// reach the remote shell as user input.
func (r *Runner) factorioPlayersListCommand(name, listFile string) *cli.Command {
	if strings.ContainsAny(listFile, " ;&|`$") {
		// Compile-time hardening: reject unexpected literals at builder
		// time rather than risking a malformed remote command.
		panic("factorio.players: refusing suspicious listFile literal: " + listFile)
	}
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Print entries from %s.", listFile),
		Action: verb.Wrap(
			verb.Spec{
				Name: "factorio.players." + name,
				Action: r.factorioRemote(fmt.Sprintf(
					"if [ -f %s/%s ]; then jq -r '.[]' %s/%s; else echo '(no %s)'; fi",
					r.factorioServerDir(), listFile,
					r.factorioServerDir(), listFile,
					listFile,
				)),
			},
			r.Audit,
		),
	}
}
