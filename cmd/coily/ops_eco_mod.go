package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// ecoModCommand wraps mod-file operations against the Eco dedicated
// server's tree on kai-server. Today only `push` exists. It mirrors the
// `invoke push-asset` tasks in eco-mods / eco-mods-public: scp a zip to
// <server_dir>, unzip -o on the server. The zip's internal layout
// determines the extract destination - callers build the zip with the
// paths they want on the server.
func (r *Runner) ecoModCommand() *cli.Command {
	return &cli.Command{
		Name:  "mod",
		Usage: "Push mod archives into the eco-server tree on kai-server.",
		Description: `mod wraps file-level operations against the Eco dedicated server on
kai-server. Today only 'push' exists; list/remove land when Kai needs
them.

A push does not restart the server - run 'coily eco restart' separately
when you want the new mod(s) to take effect.`,
		Commands: []*cli.Command{
			r.ecoModPushCommand(),
		},
	}
}

func (r *Runner) ecoModPushCommand() *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: "scp a .zip to <server_dir> on kai-server and unzip -o it.",
		Description: `push mirrors the 'invoke push-asset' tasks in eco-mods and
eco-mods-public: scp --src to <server_dir> on kai-server, then run
'unzip -o' on the server so the archive's internal paths extract in
place. The zip's top-level layout IS the install layout - callers decide
where files land by how they build the zip.

Typical archive shapes (same convention as eco-mods/tasks.py):

  ShopBoat.zip           -> Mods/UserCode/ShopBoat/*.cs     (UserCode mod)
  EcoJobsTracker.zip     -> Mods/EcoJobsTracker/*.dll       (compiled mod)

--server-dir defaults to config.eco.server_dir (embedded default:
/home/kai/Steam/steamapps/common/EcoServer). --keep-remote leaves the
uploaded .zip on kai-server after extract (default: delete it to keep
the Eco server root tidy).

This verb does NOT restart the server. 'coily eco restart' is the next
step for Mods/UserCode/ changes; compiled-DLL mods also need a restart
for Eco's ModKitPlugin to pick them up.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "src", Usage: "path to the local .zip", Required: true},
			&cli.StringFlag{Name: "server-dir", Usage: "override config.eco.server_dir"},
			&cli.BoolFlag{Name: "keep-remote", Usage: "leave the .zip on kai-server after extract"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "eco.mod.push",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--src":         c.String("src"),
						"--server-dir":  c.String("server-dir"),
						"--keep-remote": fmt.Sprint(c.Bool("keep-remote")),
					}, nil
				},
				Action: r.ecoModPushAction,
			},
			r.Audit,
		),
	}
}

// ecoModPushAction is the real work behind `coily eco mod push`. It is a
// method on Runner so tests can swap r.SSH for a fake client.
//
//nolint:gocyclo,cyclop // sequential flag validation, dry-run branches, and SSH+SFTP steps are linear and more readable inline than split across helpers.
func (r *Runner) ecoModPushAction(ctx context.Context, c *cli.Command) error {
	src := expandTilde(c.String("src"))
	if !strings.HasSuffix(strings.ToLower(src), ".zip") {
		return fmt.Errorf("eco mod push: --src must end in .zip, got %s", src)
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("eco mod push: stat --src %s: %w", src, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("eco mod push: --src %s is not a regular file", src)
	}

	serverDir := c.String("server-dir")
	if serverDir == "" {
		serverDir = r.Cfg.Eco.ServerDir
	}
	if serverDir == "" {
		return fmt.Errorf("eco mod push: pass --server-dir or set eco.server_dir in the embedded config")
	}
	if err := policy.ValidateArg("--server-dir", serverDir); err != nil {
		return fmt.Errorf("eco mod push: %w", err)
	}

	host := r.Cfg.KaiServer.TailscaleHost
	user := r.Cfg.KaiServer.SSHUser
	if host == "" || user == "" {
		return fmt.Errorf("eco mod push: kai_server.tailscale_host or ssh_user not configured")
	}

	// The archive basename is local input; validate it before interpolating
	// into the remote shell command. The shell-meta check is enough: the
	// zip sits under server_dir, not in a location where a traversal is
	// interesting, and the filename round-trips through scp's own parser.
	base := filepath.Base(src)
	if err := policy.ValidateArg("--src basename", base); err != nil {
		return fmt.Errorf("eco mod push: %w", err)
	}

	// Forward slashes - remote is Linux.
	remotePath := path.Join(serverDir, base)

	if err := r.SSH.CopyTo(ctx, host, user, src, remotePath); err != nil {
		return fmt.Errorf("eco mod push: upload: %w", err)
	}
	fmt.Fprintf(os.Stderr, "uploaded %s -> %s:%s\n", base, host, remotePath)

	extract := fmt.Sprintf("cd %s && unzip -o %s", serverDir, base)
	if _, stderr, err := r.SSH.Run(ctx, host, user, extract); err != nil {
		return fmt.Errorf("eco mod push: remote unzip: %w (stderr: %s)", err, strings.TrimSpace(stderr))
	}
	fmt.Fprintf(os.Stderr, "unzipped %s at %s:%s\n", base, host, serverDir)

	if !c.Bool("keep-remote") {
		// rm the uploaded archive so Eco's server root doesn't accumulate
		// one .zip per push. Extracted files stay in place.
		rm := fmt.Sprintf("rm -f %s", remotePath)
		if _, stderr, err := r.SSH.Run(ctx, host, user, rm); err != nil {
			return fmt.Errorf("eco mod push: remote cleanup: %w (stderr: %s)", err, strings.TrimSpace(stderr))
		}
	}

	fmt.Fprintf(os.Stderr, "eco mod push: done. Run 'coily eco restart' to load.\n")
	return nil
}
