# coily gh - full reference

Mirrors `gh`. Underlying version at scan time: gh version 2.66.1 (2025-01-31)

Command shape: `coily gh <verb...> [flags]`. Flags match the underlying CLI.

## `coily gh api`

### `coily gh api`

Makes an authenticated HTTP request to the GitHub API and prints the response.

Flags:

- `--cache`
- `--field`
- `--header`
- `--hostname`
- `--include`
- `--input`
- `--jq`
- `--method`
- `--paginate`
- `--preview`
- `--raw-field`
- `--silent`
- `--slurp`
- `--template`
- `--verbose`
- `--help`

## `coily gh issue`

### `coily gh issue` (group)

Work with GitHub issues.

Subcommands: `create`, `list`, `status`, `close`, `comment`, `delete`, `develop`, `edit`, `lock`, `pin`, `reopen`, `transfer`, `unlock`, `unpin`, `view`

### `coily gh issue create`

Create an issue on GitHub.

Flags:

- `--assignee`
- `--body`
- `--body-file`
- `--editor`
- `--label`
- `--milestone`
- `--project`
- `--recover`
- `--template`
- `--title`
- `--web`
- `--help`
- `--repo`

### `coily gh issue list`

List issues in a GitHub repository.

Flags:

- `--app`
- `--assignee`
- `--author`
- `--jq`
- `--json`
- `--label`
- `--limit`
- `--mention`
- `--milestone`
- `--search`
- `--state`
- `--template`
- `--web`
- `--help`
- `--repo`

### `coily gh issue status`

Show status of relevant issues

Flags:

- `--jq`
- `--json`
- `--template`
- `--help`
- `--repo`

### `coily gh issue close`

Close issue

Flags:

- `--comment`
- `--reason`
- `--help`
- `--repo`

### `coily gh issue comment`

Add a comment to a GitHub issue.

Flags:

- `--body`
- `--body-file`
- `--edit-last`
- `--editor`
- `--web`
- `--help`
- `--repo`

### `coily gh issue delete`

Delete issue

Flags:

- `--yes`
- `--help`
- `--repo`

### `coily gh issue develop`

Manage linked branches for an issue.

Flags:

- `--base`
- `--branch-repo`
- `--checkout`
- `--list`
- `--name`
- `--help`
- `--repo`

### `coily gh issue edit`

Edit one or more issues within the same repository.

Flags:

- `--add-assignee`
- `--add-label`
- `--add-project`
- `--body`
- `--body-file`
- `--milestone`
- `--remove-assignee`
- `--remove-label`
- `--remove-milestone`
- `--remove-project`
- `--title`
- `--help`
- `--repo`

### `coily gh issue lock`

Lock issue conversation

Flags:

- `--reason`
- `--help`
- `--repo`

### `coily gh issue pin`

Pin an issue to a repository.

Flags:

- `--help`
- `--repo`

### `coily gh issue reopen`

Reopen issue

Flags:

- `--comment`
- `--help`
- `--repo`

### `coily gh issue transfer`

Transfer issue to another repository

Flags:

- `--help`
- `--repo`

### `coily gh issue unlock`

Unlock issue conversation

Flags:

- `--help`
- `--repo`

### `coily gh issue unpin`

Unpin an issue from a repository.

Flags:

- `--help`
- `--repo`

### `coily gh issue view`

Display the title, body, and other information about an issue.

Flags:

- `--comments`
- `--jq`
- `--json`
- `--template`
- `--web`
- `--help`
- `--repo`

## `coily gh pr`

### `coily gh pr` (group)

Work with GitHub pull requests.

Subcommands: `create`, `list`, `status`, `checkout`, `checks`, `close`, `comment`, `diff`, `edit`, `lock`, `merge`, `ready`, `reopen`, `review`, `unlock`, `view`

### `coily gh pr create`

Create a pull request on GitHub.

Flags:

- `--assignee`
- `--base`
- `--body`
- `--body-file`
- `--draft`
- `--dry-run`
- `--editor`
- `--fill`
- `--fill-first`
- `--fill-verbose`
- `--head`
- `--label`
- `--milestone`
- `--no-maintainer-edit`
- `--project`
- `--recover`
- `--reviewer`
- `--template`
- `--title`
- `--web`
- `--help`
- `--repo`

### `coily gh pr list`

List pull requests in a GitHub repository.

Flags:

- `--app`
- `--assignee`
- `--author`
- `--base`
- `--draft`
- `--head`
- `--jq`
- `--json`
- `--label`
- `--limit`
- `--search`
- `--state`
- `--template`
- `--web`
- `--help`
- `--repo`

### `coily gh pr status`

Show status of relevant pull requests

Flags:

- `--conflict-status`
- `--jq`
- `--json`
- `--template`
- `--help`
- `--repo`

### `coily gh pr checkout`

Check out a pull request in git

Flags:

- `--branch`
- `--detach`
- `--force`
- `--recurse-submodules`
- `--help`
- `--repo`

### `coily gh pr checks`

Show CI status for a single pull request.

Flags:

- `--fail-fast`
- `--interval`
- `--jq`
- `--json`
- `--required`
- `--template`
- `--watch`
- `--web`
- `--help`
- `--repo`

### `coily gh pr close`

Close a pull request

Flags:

- `--comment`
- `--delete-branch`
- `--help`
- `--repo`

### `coily gh pr comment`

Add a comment to a GitHub pull request.

Flags:

- `--body`
- `--body-file`
- `--edit-last`
- `--editor`
- `--web`
- `--help`
- `--repo`

### `coily gh pr diff`

View changes in a pull request.

Flags:

- `--color`
- `--name-only`
- `--patch`
- `--web`
- `--help`
- `--repo`

### `coily gh pr edit`

Edit a pull request.

Flags:

- `--add-assignee`
- `--add-label`
- `--add-project`
- `--add-reviewer`
- `--base`
- `--body`
- `--body-file`
- `--milestone`
- `--remove-assignee`
- `--remove-label`
- `--remove-milestone`
- `--remove-project`
- `--remove-reviewer`
- `--title`
- `--help`
- `--repo`

### `coily gh pr lock`

Lock pull request conversation

Flags:

- `--reason`
- `--help`
- `--repo`

### `coily gh pr merge`

Merge a pull request on GitHub.

Flags:

- `--admin`
- `--author-email`
- `--auto`
- `--body`
- `--body-file`
- `--delete-branch`
- `--disable-auto`
- `--match-head-commit`
- `--merge`
- `--rebase`
- `--squash`
- `--subject`
- `--help`
- `--repo`

### `coily gh pr ready`

Mark a pull request as ready for review.

Flags:

- `--undo`
- `--help`
- `--repo`

### `coily gh pr reopen`

Reopen a pull request

Flags:

- `--comment`
- `--help`
- `--repo`

### `coily gh pr review`

Add a review to a pull request.

Flags:

- `--approve`
- `--body`
- `--body-file`
- `--comment`
- `--request-changes`
- `--help`
- `--repo`

### `coily gh pr unlock`

Unlock pull request conversation

Flags:

- `--help`
- `--repo`

### `coily gh pr view`

Display the title, body, and other information about a pull request.

Flags:

- `--comments`
- `--jq`
- `--json`
- `--template`
- `--web`
- `--help`
- `--repo`

## `coily gh release`

### `coily gh release` (group)

Manage releases

Subcommands: `create`, `list`, `delete`, `download`, `edit`, `upload`, `view`

### `coily gh release create`

Create a new GitHub Release for a repository.

Flags:

- `--discussion-category`
- `--draft`
- `--generate-notes`
- `--latest`
- `--notes`
- `--notes-file`
- `--notes-from-tag`
- `--notes-start-tag`
- `--prerelease`
- `--target`
- `--title`
- `--verify-tag`
- `--help`
- `--repo`

### `coily gh release list`

List releases in a repository

Flags:

- `--exclude-drafts`
- `--exclude-pre-releases`
- `--jq`
- `--json`
- `--limit`
- `--order`
- `--template`
- `--help`
- `--repo`

### `coily gh release delete`

Delete a release

Flags:

- `--cleanup-tag`
- `--yes`
- `--help`
- `--repo`

### `coily gh release download`

Download assets from a GitHub release.

Flags:

- `--archive`
- `--clobber`
- `--dir`
- `--output`
- `--pattern`
- `--skip-existing`
- `--help`
- `--repo`

### `coily gh release edit`

Edit a release

Flags:

- `--discussion-category`
- `--draft`
- `--latest`
- `--notes`
- `--notes-file`
- `--prerelease`
- `--tag`
- `--target`
- `--title`
- `--verify-tag`
- `--help`
- `--repo`

### `coily gh release upload`

Upload asset files to a GitHub Release.

Flags:

- `--clobber`
- `--help`
- `--repo`

### `coily gh release view`

View information about a GitHub Release.

Flags:

- `--jq`
- `--json`
- `--template`
- `--web`
- `--help`
- `--repo`

## `coily gh repo`

### `coily gh repo` (group)

Work with GitHub repositories.

Subcommands: `create`, `list`, `archive`, `autolink`, `clone`, `delete`, `deploy-key`, `edit`, `fork`, `gitignore`, `license`, `rename`, `sync`, `unarchive`, `view`

### `coily gh repo create`

Create a new GitHub repository.

Flags:

- `--add-readme`
- `--clone`
- `--description`
- `--disable-issues`
- `--disable-wiki`
- `--gitignore`
- `--homepage`
- `--include-all-branches`
- `--internal`
- `--license`
- `--private`
- `--public`
- `--push`
- `--remote`
- `--source`
- `--team`
- `--template`
- `--help`

### `coily gh repo list`

List repositories owned by a user or organization.

Flags:

- `--archived`
- `--fork`
- `--jq`
- `--json`
- `--language`
- `--limit`
- `--no-archived`
- `--source`
- `--template`
- `--topic`
- `--visibility`
- `--help`

### `coily gh repo archive`

Archive a GitHub repository.

Flags:

- `--yes`
- `--help`

### `coily gh repo autolink` (group)

Autolinks link issues, pull requests, commit messages, and release descriptions to external third-party services.

Subcommands: `create`, `list`, `view`

### `coily gh repo autolink create`

Create a new autolink reference for a repository.

Flags:

- `--numeric`
- `--help`
- `--repo`

### `coily gh repo autolink list`

Gets all autolink references that are configured for a repository.

Flags:

- `--jq`
- `--json`
- `--template`
- `--web`
- `--help`
- `--repo`

### `coily gh repo autolink view`

View an autolink reference for a repository.

Flags:

- `--jq`
- `--json`
- `--template`
- `--help`
- `--repo`

### `coily gh repo clone`

Clone a GitHub repository locally.

Flags:

- `--upstream-remote-name`
- `--help`

### `coily gh repo delete`

Delete a GitHub repository.

Flags:

- `--yes`
- `--help`

### `coily gh repo deploy-key` (group)

Manage deploy keys in a repository

Subcommands: `add`, `delete`, `list`

### `coily gh repo deploy-key add`

Add a deploy key to a GitHub repository.

Flags:

- `--allow-write`
- `--title`
- `--help`
- `--repo`

### `coily gh repo deploy-key delete`

Delete a deploy key from a GitHub repository

Flags:

- `--help`
- `--repo`

### `coily gh repo deploy-key list`

List deploy keys in a GitHub repository

Flags:

- `--jq`
- `--json`
- `--template`
- `--help`
- `--repo`

### `coily gh repo edit`

Edit repository settings.

Flags:

- `--accept-visibility-change-consequences`
- `--add-topic`
- `--allow-forking`
- `--allow-update-branch`
- `--default-branch`
- `--delete-branch-on-merge`
- `--description`
- `--enable-advanced-security`
- `--enable-auto-merge`
- `--enable-discussions`
- `--enable-issues`
- `--enable-merge-commit`
- `--enable-projects`
- `--enable-rebase-merge`
- `--enable-secret-scanning`
- `--enable-secret-scanning-push-protection`
- `--enable-squash-merge`
- `--enable-wiki`
- `--homepage`
- `--remove-topic`
- `--template`
- `--visibility`
- `--help`

### `coily gh repo fork`

Create a fork of a repository.

Flags:

- `--clone`
- `--default-branch-only`
- `--fork-name`
- `--org`
- `--remote`
- `--remote-name`
- `--help`

### `coily gh repo gitignore` (group)

List and view available repository gitignore templates

Subcommands: `list`, `view`

### `coily gh repo gitignore list`

List available repository gitignore templates

Flags:

- `--help`

### `coily gh repo gitignore view`

View an available repository `.gitignore` template.

Flags:

- `--help`

### `coily gh repo license` (group)

Explore repository licenses

Subcommands: `list`, `view`

### `coily gh repo license list`

List common repository licenses.

Flags:

- `--help`

### `coily gh repo license view`

View a specific repository license by license key or SPDX ID.

Flags:

- `--web`
- `--help`

### `coily gh repo rename`

Rename a GitHub repository.

Flags:

- `--repo`
- `--yes`
- `--help`

### `coily gh repo sync`

Sync destination repository from source repository.

Flags:

- `--branch`
- `--force`
- `--source`
- `--help`

### `coily gh repo unarchive`

Unarchive a GitHub repository.

Flags:

- `--yes`
- `--help`

### `coily gh repo view`

Display the description and the README of a GitHub repository.

Flags:

- `--branch`
- `--jq`
- `--json`
- `--template`
- `--web`
- `--help`

## `coily gh run`

### `coily gh run` (group)

List, view, and watch recent workflow runs from GitHub Actions.

Subcommands: `cancel`, `delete`, `download`, `list`, `rerun`, `view`, `watch`

### `coily gh run cancel`

Cancel a workflow run

Flags:

- `--help`
- `--repo`

### `coily gh run delete`

Delete a workflow run

Flags:

- `--help`
- `--repo`

### `coily gh run download`

Download artifacts generated by a GitHub Actions workflow run.

Flags:

- `--dir`
- `--name`
- `--pattern`
- `--help`
- `--repo`

### `coily gh run list`

List recent workflow runs.

Flags:

- `--all`
- `--branch`
- `--commit`
- `--created`
- `--event`
- `--jq`
- `--json`
- `--limit`
- `--status`
- `--template`
- `--user`
- `--workflow`
- `--help`
- `--repo`

### `coily gh run rerun`

Rerun an entire run, only failed jobs, or a specific job from a run.

Flags:

- `--debug`
- `--failed`
- `--job`
- `--help`
- `--repo`

### `coily gh run view`

View a summary of a workflow run.

Flags:

- `--attempt`
- `--exit-status`
- `--job`
- `--jq`
- `--json`
- `--log`
- `--log-failed`
- `--template`
- `--verbose`
- `--web`
- `--help`
- `--repo`

### `coily gh run watch`

Watch a run until it completes, showing its progress.

Flags:

- `--exit-status`
- `--interval`
- `--help`
- `--repo`

## `coily gh search`

### `coily gh search` (group)

Search across all of GitHub.

Subcommands: `code`, `commits`, `issues`, `prs`, `repos`

### `coily gh search code`

Search within code in GitHub repositories.

Flags:

- `--extension`
- `--filename`
- `--jq`
- `--json`
- `--language`
- `--limit`
- `--match`
- `--owner`
- `--repo`
- `--size`
- `--template`
- `--web`
- `--help`

### `coily gh search commits`

Search for commits on GitHub.

Flags:

- `--author`
- `--author-date`
- `--author-email`
- `--author-name`
- `--committer`
- `--committer-date`
- `--committer-email`
- `--committer-name`
- `--hash`
- `--jq`
- `--json`
- `--limit`
- `--merge`
- `--order`
- `--owner`
- `--parent`
- `--repo`
- `--sort`
- `--template`
- `--tree`
- `--visibility`
- `--web`
- `--help`

### `coily gh search issues`

Search for issues on GitHub.

Flags:

- `--app`
- `--archived`
- `--assignee`
- `--author`
- `--closed`
- `--commenter`
- `--comments`
- `--created`
- `--include-prs`
- `--interactions`
- `--involves`
- `--jq`
- `--json`
- `--label`
- `--language`
- `--limit`
- `--locked`
- `--match`
- `--mentions`
- `--milestone`
- `--no-assignee`
- `--no-label`
- `--no-milestone`
- `--no-project`
- `--order`
- `--owner`
- `--project`
- `--reactions`
- `--repo`
- `--sort`
- `--state`
- `--team-mentions`
- `--template`
- `--updated`
- `--visibility`
- `--web`
- `--help`

### `coily gh search prs`

Search for pull requests on GitHub.

Flags:

- `--app`
- `--archived`
- `--assignee`
- `--author`
- `--base`
- `--checks`
- `--closed`
- `--commenter`
- `--comments`
- `--created`
- `--draft`
- `--head`
- `--interactions`
- `--involves`
- `--jq`
- `--json`
- `--label`
- `--language`
- `--limit`
- `--locked`
- `--match`
- `--mentions`
- `--merged`
- `--merged-at`
- `--milestone`
- `--no-assignee`
- `--no-label`
- `--no-milestone`
- `--no-project`
- `--order`
- `--owner`
- `--project`
- `--reactions`
- `--repo`
- `--review`
- `--review-requested`
- `--reviewed-by`
- `--sort`
- `--state`
- `--team-mentions`
- `--template`
- `--updated`
- `--visibility`
- `--web`
- `--help`

### `coily gh search repos`

Search for repositories on GitHub.

Flags:

- `--archived`
- `--created`
- `--followers`
- `--forks`
- `--good-first-issues`
- `--help-wanted-issues`
- `--include-forks`
- `--jq`
- `--json`
- `--language`
- `--license`
- `--limit`
- `--match`
- `--number-topics`
- `--order`
- `--owner`
- `--size`
- `--sort`
- `--stars`
- `--template`
- `--topic`
- `--updated`
- `--visibility`
- `--web`
- `--help`

## `coily gh secret`

### `coily gh secret` (group)

Secrets can be set at the repository, or organization level for use in

Subcommands: `delete`, `list`, `set`

### `coily gh secret delete`

Delete a secret on one of the following levels:

Flags:

- `--app`
- `--env`
- `--org`
- `--user`
- `--help`
- `--repo`

### `coily gh secret list`

List secrets on one of the following levels:

Flags:

- `--app`
- `--env`
- `--jq`
- `--json`
- `--org`
- `--template`
- `--user`
- `--help`
- `--repo`

### `coily gh secret set`

Set a value for a secret on one of the following levels:

Flags:

- `--app`
- `--body`
- `--env`
- `--env-file`
- `--no-store`
- `--org`
- `--repos`
- `--user`
- `--visibility`
- `--help`
- `--repo`

## `coily gh workflow`

### `coily gh workflow` (group)

List, view, and run workflows in GitHub Actions.

Subcommands: `disable`, `enable`, `list`, `run`, `view`

### `coily gh workflow disable`

Disable a workflow, preventing it from running or showing up when listing workflows.

Flags:

- `--help`
- `--repo`

### `coily gh workflow enable`

Enable a workflow, allowing it to be run and show up when listing workflows.

Flags:

- `--help`
- `--repo`

### `coily gh workflow list`

List workflow files, hiding disabled workflows by default.

Flags:

- `--all`
- `--jq`
- `--json`
- `--limit`
- `--template`
- `--help`
- `--repo`

### `coily gh workflow run`

Create a `workflow_dispatch` event for a given workflow.

Flags:

- `--field`
- `--json`
- `--raw-field`
- `--ref`
- `--help`
- `--repo`

### `coily gh workflow view`

View the summary of a workflow

Flags:

- `--ref`
- `--web`
- `--yaml`
- `--help`
- `--repo`

