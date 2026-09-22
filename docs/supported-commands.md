# Supported commands

Which CLI help outputs jym can extract subcommands from. Generated from
`testdata/help/manifest.tsv` and the fixtures under `testdata/help/` by
`go run ./cmd/jym-help-report` — do not edit by hand; change the manifest and
regenerate instead. Checked rows are asserted by `go test ./internal/helptext`.

Status: **ok** parses fully, **partial** parses a known subset, **leaf**
has no documented subcommands (empty is correct), **empty** currently
yields nothing, **bogus** extracts words that are not subcommands,
**todo** is tracked for support, **out-of-scope** will not be supported.

- [x] `apt` — ok (7 parsed; dash-separated entries)
- **aws**
  - [x] `aws` — ok (302 parsed; nroff overstrike + o bullets)
  - [x] `aws s3` — ok (9 parsed; o bullets under AVAILABLE COMMANDS)
  - [x] `aws ec2` — ok (520 parsed; o bullets under AVAILABLE COMMANDS)
  - [x] `aws iam` — ok (160 parsed; o bullets under AVAILABLE COMMANDS)
- [x] `az` — ok (100 parsed)
- [x] `brew` — ok (9 parsed; tool-prefixed usage lines)
- [x] `bun` — ok (25 parsed)
- **cargo**
  - [x] `cargo` — ok (16 parsed)
  - [x] `cargo install` — leaf (leaf command, no subcommands)
- [ ] `curl` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `deno` — ok (48 parsed)
- **docker**
  - [x] `docker` — ok (43 parsed)
  - [x] `docker compose` — ok (35 parsed)
  - [x] `docker image` — ok (12 parsed)
  - [x] `docker container` — ok (25 parsed)
  - [x] `docker network` — ok (7 parsed)
  - [x] `docker volume` — ok (5 parsed)
- [ ] `gcloud` — todo (fixture not committed; not captured)
- **gh**
  - [x] `gh` — ok (36 parsed)
  - [x] `gh pr` — ok (17 parsed; prose headings rejected)
  - [x] `gh repo` — ok (16 parsed; prose headings rejected)
  - [x] `gh auth` — ok (7 parsed)
  - [x] `gh issue` — ok (15 parsed)
  - [x] `gh run` — ok (7 parsed)
- **git**
  - [x] `git` — ok (17 parsed; help_args: help -a)
  - [x] `git remote` — ok (5 parsed; man-page COMMANDS section)
  - [x] `git stash` — ok (7 parsed; man-page COMMANDS section)
  - [x] `git submodule` — ok (7 parsed; man-page COMMANDS section)
  - [x] `git worktree` — ok (8 parsed; man-page COMMANDS section)
  - [x] `git bisect` — ok (9 parsed; man-page COMMANDS section)
  - [x] `git config` — leaf (env-var sections are skipped)
- **go**
  - [x] `go` — ok (37 parsed)
  - [x] `go mod` — ok (8 parsed)
  - [x] `go work` — ok (5 parsed)
- [ ] `gradle` — out-of-scope (fixture not committed; goals defined by plugins, not in --help)
- **helm**
  - [x] `helm` — ok (26 parsed)
  - [x] `helm repo` — ok (5 parsed)
- [ ] `jq` — out-of-scope (fixture not committed; no subcommand concept)
- **kubectl**
  - [x] `kubectl` — ok (42 parsed)
  - [x] `kubectl config` — ok (15 parsed)
  - [x] `kubectl rollout` — ok (6 parsed)
- [ ] `make` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `mise` — ok (66 parsed)
- [ ] `mvn` — out-of-scope (goals defined by plugins, not in --help)
- **npm**
  - [x] `npm` — ok (15 parsed; comma inventory format)
  - [ ] `npm cache` — empty (fixture not committed; npm-prefixed synopsis lines unsupported)
- **pip**
  - [x] `pip` — ok (16 parsed)
  - [x] `pip install` — leaf (leaf command, no subcommands)
- [x] `pnpm` — ok (92 parsed)
- [x] `poetry` — ok (20 parsed; help_args: list)
- [ ] `python3` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `rustup` — ok (18 parsed)
- [ ] `ssh` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `sudo` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `systemctl` — ok (13 parsed; [ARG...] placeholders skipped)
- [x] `tar` — leaf (option-value sections are skipped)
- **terraform**
  - [x] `terraform` — ok (fixture not committed; parses; fixture not committed (BUSL))
  - [x] `terraform state` — ok (fixture not committed; parses; fixture not committed (BUSL))
- [ ] `tmux` — empty ("list-commands" name (alias) form unsupported; help_args: list-commands)
- [x] `uv` — ok (23 parsed)
- [x] `yarn` — ok (27 parsed; tool-prefixed usage lines)

## Totals

- ok: 47
- leaf: 4
- empty: 2
- todo: 1
- out-of-scope: 8
