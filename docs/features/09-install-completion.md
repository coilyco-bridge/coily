# 9. `coily install-completion`

**What it is**: Writes a shell completion script to the user's home dir.

**How to invoke**:
- `coily install-completion` - auto-detect shell
- `coily install-completion --shell bash|zsh|fish`
- `coily install-completion --dry-run` - print instead of writing

**Expected shape**: Writes `~/.local/share/coily/completion.<shell>` (bash/zsh) or `~/.config/fish/completions/coily.fish`. Prints source instruction to stderr.

**Test prompt**:
> Verify `coily install-completion --shell zsh --dry-run` prints a script containing `compdef _coily_zsh_autocomplete coily`. Verify `coily install-completion --shell bash` (without --dry-run) writes `~/.local/share/coily/completion.bash` and the file is non-empty bash. Verify after sourcing the bash completion, `complete -p coily` shows the completion is registered. Clean up the file when done.
