package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/policy"
	"github.com/urfave/cli/v3"
)

// ecoModCommand wraps mod-file operations against the Eco dedicated
// server's tree on kai-server. Today only `push` exists. It mirrors the
// `invoke push-asset` tasks in eco-mods / eco-mods-public: copy a zip into
// <server_dir>, unzip -o it. The SSH upload path was removed, so this runs
// on kai-server itself. The zip's internal layout determines the extract
// destination - callers build the zip with the paths they want.
func (r *Runner) ecoModCommand() *cli.Command {
	return &cli.Command{
		Name:  "mod",
		Usage: "Push mod archives into the eco-server tree on kai-server.",
		Description: `mod wraps file-level operations against the Eco dedicated server on
kai-server. Today only 'push' exists; list/remove land when Kai needs
them.

A push does not restart the server - run 'coily gaming eco restart' separately
when you want the new mod(s) to take effect.`,
		Commands: []*cli.Command{
			r.ecoModPushCommand(),
		},
	}
}

func (r *Runner) ecoModPushCommand() *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: "copy a .zip into <server_dir> and unzip -o it (run on kai-server).",
		Description: `push mirrors the 'invoke push-asset' tasks in eco-mods and
eco-mods-public: copy --src into <server_dir>, then run 'unzip -o' so the
archive's internal paths extract in place. The zip's top-level layout IS
the install layout - callers decide where files land by how they build
the zip. Runs on kai-server itself; the SSH upload path was removed.

Typical archive shapes (same convention as eco-mods/tasks.py):

  ShopBoat.zip           -> Mods/UserCode/ShopBoat/*.cs     (UserCode mod)
  EcoJobsTracker.zip     -> Mods/EcoJobsTracker/*.dll       (compiled mod)

--server-dir defaults to config.eco.server_dir (embedded default:
/home/kai/Steam/steamapps/common/EcoServer). --keep-remote leaves the
uploaded .zip on kai-server after extract (default: delete it to keep
the Eco server root tidy).

This verb does NOT restart the server. 'coily gaming eco restart' is the next
step for Mods/UserCode/ changes; compiled-DLL mods also need a restart
for Eco's ModKitPlugin to pick them up.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "src", Usage: "path to the local .zip", Required: true},
			&cli.StringFlag{Name: "server-dir", Usage: "override config.eco.server_dir"},
			&cli.BoolFlag{Name: "keep-remote", Usage: "leave the .zip on kai-server after extract"},
		},
		Action: r.WrapVerb(
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

// ecoModPushAction is the real work behind `coily eco mod push`. The SSH
// transport that used to scp the archive to kai-server was removed, so this
// runs only on kai-server itself (hostIsLocal): it copies the local .zip
// into <server_dir> and unzips it in place. A non-local configured host
// returns errRemoteRemoved.
//
//nolint:gocyclo,cyclop // sequential flag validation, host gate, and copy+unzip+cleanup steps are linear and more readable inline than split across helpers.
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
	if host == "" {
		return fmt.Errorf("eco mod push: kai_server.tailscale_host not configured")
	}
	if !hostIsLocal(host) {
		return errRemoteRemoved("eco mod push", host)
	}

	// The archive basename is operator input; validate it before joining it
	// into the destination path under server_dir.
	base := filepath.Base(src)
	if err := policy.ValidateArg("--src basename", base); err != nil {
		return fmt.Errorf("eco mod push: %w", err)
	}

	destPath := filepath.Join(serverDir, base)
	if err := copyFile(src, destPath); err != nil {
		return fmt.Errorf("eco mod push: copy into server dir: %w", err)
	}
	fmt.Fprintf(os.Stderr, "copied %s -> %s\n", base, destPath)

	// -d extracts into server_dir; the archive's internal paths land in place.
	if err := r.Runner.Exec(ctx, "unzip", "-o", destPath, "-d", serverDir); err != nil {
		return fmt.Errorf("eco mod push: unzip: %w", err)
	}
	fmt.Fprintf(os.Stderr, "unzipped %s into %s\n", base, serverDir)

	if !c.Bool("keep-remote") {
		// Drop the copied archive so Eco's server root doesn't accumulate
		// one .zip per push. Extracted files stay in place.
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("eco mod push: cleanup %s: %w", destPath, err)
		}
	}

	fmt.Fprintf(os.Stderr, "eco mod push: done. Run 'coily gaming eco restart' to load.\n")
	return nil
}

// copyFile copies src to dst, creating or truncating dst with 0o644. Used by
// eco mod push to place a local archive into the eco server dir. The named
// return surfaces a close error on the write side without masking a copy error.
func copyFile(src, dst string) (err error) {
	// #nosec G304 -- src is an operator-supplied .zip path, already stat'd as a
	// regular file; dst is server_dir/base with both segments policy-validated.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck,gosec // read side; copy error dominates.
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	_, err = io.Copy(out, in)
	return err
}
