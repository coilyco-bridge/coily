package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

func (r *Runner) installCompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "install-completion",
		Usage: "Install shell tab-completion for coily.",
		Description: `install-completion writes a small sourceable file that wires coily into the
user's shell completion system. Supports bash, zsh, and fish. Shell is
auto-detected from $SHELL unless --shell is passed.

By default writes to:
  bash:  ~/.local/share/coily/completion.bash
  zsh:   ~/.local/share/coily/completion.zsh
  fish:  ~/.config/fish/completions/coily.fish

After running, follow the printed instructions to source the file from your
shell rc (or for fish, restart the shell).

Pass --dry-run to print the script to stdout instead of writing.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "shell",
				Usage: "one of bash, zsh, fish. Default: auto-detect from $SHELL",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the completion script to stdout instead of writing a file",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:      "install-completion",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--shell": c.String("shell")}, nil
				},
				Action: installCompletionAction,
			},
			r.Audit,
		),
	}
}

func installCompletionAction(_ context.Context, c *cli.Command) error {
	shell := c.String("shell")
	if shell == "" {
		shell = detectShell()
	}
	if shell == "" {
		return fmt.Errorf("install-completion: could not detect shell from $SHELL. Pass --shell bash|zsh|fish")
	}

	script, path, srcInstr, err := completionFor(shell)
	if err != nil {
		return err
	}

	if c.Bool("dry-run") {
		fmt.Print(script)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("install-completion: mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		return fmt.Errorf("install-completion: write %s: %w", path, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n\n", path)
	fmt.Fprintln(os.Stderr, "To activate:")
	fmt.Fprintln(os.Stderr, srcInstr)
	return nil
}

func detectShell() string {
	s := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(s, "/bash"):
		return "bash"
	case strings.HasSuffix(s, "/zsh"):
		return "zsh"
	case strings.HasSuffix(s, "/fish"):
		return "fish"
	}
	return ""
}

func completionFor(shell string) (script, path, instructions string, err error) {
	home, herr := os.UserHomeDir()
	if herr != nil {
		return "", "", "", fmt.Errorf("install-completion: home dir: %w", herr)
	}
	switch shell {
	case "bash":
		return completionBash(), filepath.Join(home, ".local", "share", "coily", "completion.bash"),
			"  echo 'source " + filepath.Join(home, ".local/share/coily/completion.bash") + "' >> ~/.bashrc", nil
	case "zsh":
		return completionZsh(), filepath.Join(home, ".local", "share", "coily", "completion.zsh"),
			"  echo 'source " + filepath.Join(home, ".local/share/coily/completion.zsh") + "' >> ~/.zshrc", nil
	case "fish":
		return completionFish(), filepath.Join(home, ".config", "fish", "completions", "coily.fish"),
			"  (fish auto-loads files in ~/.config/fish/completions/. Restart your shell.)", nil
	}
	return "", "", "", fmt.Errorf("install-completion: unknown shell %q (want bash, zsh, or fish)", shell)
}

func completionBash() string {
	return `# coily bash completion. urfave/cli v3 style.
_coily_bash_autocomplete() {
  local cur opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ "$cur" == "-"* ]]; then
    opts=$( "${COMP_WORDS[@]:0:$COMP_CWORD}" "${cur}" --generate-shell-completion )
  else
    opts=$( "${COMP_WORDS[@]:0:$COMP_CWORD}" --generate-shell-completion )
  fi
  COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
  return 0
}
complete -o bashdefault -o default -F _coily_bash_autocomplete coily
`
}

func completionZsh() string {
	return `# coily zsh completion. urfave/cli v3 style.
_coily_zsh_autocomplete() {
  local -a opts
  local cur
  cur="${words[-1]}"
  if [[ "$cur" == "-"* ]]; then
    opts=( $("${words[@]:0:#words[@]-1}" "${cur}" --generate-shell-completion) )
  else
    opts=( $("${words[@]:0:#words[@]-1}" --generate-shell-completion) )
  fi
  if [[ ${#opts[@]} -eq 0 ]]; then
    _path_files
    return
  fi
  _describe 'values' opts
  return 0
}
compdef _coily_zsh_autocomplete coily
`
}

func completionFish() string {
	return `# coily fish completion.
function __coily_complete
    set -l tokens (commandline -opc)
    set -l cur (commandline -ct)
    if test -z "$cur"
        set -l args $tokens --generate-shell-completion
    else
        set -l args $tokens $cur --generate-shell-completion
    end
    eval $args
end
complete -c coily -f -a '(__coily_complete)'
`
}
