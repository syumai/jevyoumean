# Supported commands

Which CLI help outputs jym can extract subcommands from. Generated from
`testdata/help/manifest.tsv` and the fixtures under `testdata/help/` by
`go run ./cmd/jym-help-report` — do not edit by hand; change the manifest and
regenerate instead. Checked rows are asserted by `go test ./internal/helptext`.

Status: **ok** parses fully, **partial** parses a known subset, **leaf**
has no documented subcommands (empty is correct), **empty** currently
yields nothing, **bogus** extracts words that are not subcommands,
**todo** is tracked for support, **out-of-scope** will not be supported.

- [ ] `apt` — empty (dash-separated entries not recognized)
- **aws**
  - [ ] `aws` — bogus (2 parsed; prose parsed as commands)
  - [ ] `aws s3` — bogus (26 parsed; prose parsed as commands)
  - [ ] `aws ec2` — bogus (3 parsed; prose parsed as commands)
  - [ ] `aws iam` — todo (fixture not committed; not captured)
- [ ] `az` — todo (fixture not committed; not captured)
- [ ] `brew` — todo (fixture not committed; not captured)
- [ ] `bun` — todo (fixture not committed; not captured)
- **cargo**
  - [x] `cargo` — ok (16 parsed)
  - [x] `cargo install` — leaf (leaf command, no subcommands)
- [ ] `curl` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `deno` — todo (fixture not committed; not captured)
- **docker**
  - [x] `docker` — ok (43 parsed)
  - [x] `docker compose` — ok (35 parsed)
  - [x] `docker image` — ok (12 parsed)
  - [x] `docker container` — ok (25 parsed)
  - [ ] `docker network` — todo (fixture not committed; not captured)
  - [ ] `docker volume` — todo (fixture not committed; not captured)
- [ ] `gcloud` — todo (fixture not committed; not captured)
- **gh**
  - [x] `gh` — ok (36 parsed)
  - [x] `gh pr` — partial (31 parsed; prose commas create junk names)
  - [x] `gh repo` — partial (30 parsed; prose commas create junk names)
  - [x] `gh auth` — ok (7 parsed)
  - [ ] `gh issue` — todo (fixture not committed; not captured)
  - [ ] `gh run` — todo (fixture not committed; not captured)
- **git**
  - [x] `git` — ok (17 parsed; help_args: help -a)
  - [ ] `git remote` — empty (man-page COMMANDS section broken by prose line)
  - [ ] `git stash` — empty (man-page COMMANDS section broken by prose line)
  - [ ] `git submodule` — empty (fixture not committed; not captured)
  - [ ] `git worktree` — empty (fixture not committed; not captured)
  - [ ] `git bisect` — empty (fixture not committed; not captured)
  - [ ] `git config` — bogus (2 parsed; env vars parsed as subcommands)
- **go**
  - [x] `go` — ok (19 parsed)
  - [x] `go mod` — ok (8 parsed)
  - [ ] `go work` — todo (fixture not committed; not captured)
- [ ] `gradle` — todo (fixture not committed; not captured)
- **helm**
  - [ ] `helm` — todo (fixture not committed; not captured)
  - [ ] `helm repo` — todo (fixture not committed; not captured)
- [ ] `jq` — out-of-scope (fixture not committed; no subcommand concept)
- **kubectl**
  - [ ] `kubectl` — todo (fixture not committed; not captured)
  - [ ] `kubectl config` — todo (fixture not committed; not captured)
  - [ ] `kubectl rollout` — todo (fixture not committed; not captured)
- [ ] `make` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `mise` — todo (fixture not committed; not captured)
- [ ] `mvn` — todo (fixture not committed; not captured)
- **npm**
  - [x] `npm` — ok (15 parsed; comma inventory format)
  - [ ] `npm cache` — todo (fixture not committed; not captured)
- **pip**
  - [x] `pip` — ok (16 parsed)
  - [ ] `pip install` — bogus (3 parsed; description prose parsed as commands)
- [ ] `pnpm` — todo (fixture not committed; not captured)
- [ ] `poetry` — todo (fixture not committed; not captured)
- [ ] `python3` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `rustup` — todo (fixture not committed; not captured)
- [ ] `ssh` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `sudo` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `systemctl` — partial (6 parsed; entries with [ARG...] placeholders are skipped)
- [ ] `tar` — bogus (5 parsed; option-value keywords parsed as commands)
- **terraform**
  - [ ] `terraform` — todo (fixture not committed; not captured)
  - [ ] `terraform state` — todo (fixture not committed; not captured)
- [ ] `tmux` — todo (fixture not committed; not captured)
- [ ] `uv` — todo (fixture not committed; not captured)
- [ ] `yarn` — todo (fixture not committed; not captured)

## Totals

- ok: 12
- partial: 3
- leaf: 1
- empty: 6
- bogus: 6
- todo: 28
- out-of-scope: 6
