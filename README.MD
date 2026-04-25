# Commitizen-go
> Command line utility to standardize git commit messages and manage git-flow, golang version.

![demo](docs/images/demo.gif)

Contents
- Getting Started
- Usage
- Configure

## Getting Started
### installation with Homebrew:
```
$ brew tap lintingzhen/tap
$ brew install commitizen-go
$ sudo commitizen-go install
```
### installation with AUR package (Arch Linux):
```
$ yay commitizen-go
```
### installation with source code:
```
$ make && make install
```
or
```
$ make && ./commitizen-go install
```

## Usage

### Commit
```
$ git cz commit
```

```
Usage:
  commitizen-go commit [flags]

Flags:
  -a, --all             stage all tracked modified/deleted files before committing
      --allow-empty     allow a commit with no changes
      --amend           replace the tip of the current branch
      --author string   override commit author as "Name <email>"
  -h, --help            help for commit
  -n, --no-verify       bypass pre-commit and commit-msg hooks
  -s, --signoff         add Signed-off-by trailer to the commit message
```

If any commit flag is passed, the options page of the TUI form is skipped and the flags are used directly.

### Issue (coming soon)
```
$ git cz issue
```

Starts a new piece of work: enter an issue ID, title, and type, then a properly named branch is created and checked out automatically. Branch state is tracked in `.git/git-cz.db`.

### All commands
```
Usage:
  commitizen-go [command]

Available Commands:
  commit      Record changes to the repository
  install     Install this tool to git-core as git-cz
  issue       Start work on an issue (pick issue → create branch)
  version     Print version information and quit

Flags:
  -d, --debug   debug mode, output debug info to debug.log
  -h, --help    help for commitizen-go
```

## Configure

Config file: `.git-czrc` (JSON) at repository root or `$HOME`. Repository root takes priority over home directory. You can also add the `.json` extension (`.git-czrc.json`).

### Commit message

The format is the same as the default config in [`commit/defaultConfig.go`](https://github.com/lintingzhen/commitizen-go/blob/master/commit/defaultConfig.go).

Type item:
```json
{
  "name": "type",
  "desc": "Select the type of change that you're committing:",
  "form": "select",
  "options": [
    { "name": "feat",     "desc": "feat: A new feature" },
    { "name": "fix",      "desc": "fix: A bug fix" },
    { "name": "docs",     "desc": "docs: Documentation only changes" },
    { "name": "style",    "desc": "style: Changes that do not affect the meaning of the code" },
    { "name": "refactor", "desc": "refactor: A code change that neither fixes a bug nor adds a feature" },
    { "name": "perf",     "desc": "perf: A code change that improves performance" },
    { "name": "test",     "desc": "test: Adding missing tests" },
    { "name": "chore",    "desc": "chore: Changes to the build process or auxiliary tools" },
    { "name": "revert",   "desc": "revert: Revert to a commit" },
    { "name": "WIP",      "desc": "WIP: Work in progress" }
  ],
  "required": true
}
```

Message template:
```json
"template": "{{.type}}{{with .scope}}({{.}}){{end}}: {{.subject}}{{with .body}}\n\n{{.}}{{end}}{{with .footer}}\n\n{{.}}{{end}}"
```

### Branch naming

Branch names follow the format `{issue-id}@{type}@{slugified-title}@{short-uuid}`, e.g.:

```
ABC-42@feat@add-oauth-login@550e8400
```

To override the base branch (default: auto-detected from `origin/HEAD`, then `main`, then `master`):

```json
{
  "branch": {
    "base": "develop"
  }
}
```
