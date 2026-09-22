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
  - [x] `aws s3api` — ok (94 parsed; o bullets under AVAILABLE COMMANDS)
  - [x] `aws dynamodb` — ok (52 parsed; o bullets under AVAILABLE COMMANDS)
  - [x] `aws sts` — ok (9 parsed; o bullets under AVAILABLE COMMANDS)
- [x] `az` — ok (100 parsed)
- [x] `brew` — ok (9 parsed; tool-prefixed usage lines)
- **bun**
  - [x] `bun` — ok (25 parsed)
  - [x] `bun pm` — ok (14 parsed; 2-token prefixed entries with gap desc)
- **cargo**
  - [x] `cargo` — ok (16 parsed)
  - [x] `cargo install` — leaf (leaf command, no subcommands)
  - [x] `cargo doc` — leaf (leaf command, no subcommands)
- [ ] `curl` — out-of-scope (fixture not committed; no subcommand concept)
- **deno**
  - [x] `deno` — ok (48 parsed)
  - [x] `deno jupyter` — leaf (leaf command, no subcommands)
  - [x] `deno task` — leaf (leaf command, no subcommands)
  - [x] `deno publish` — leaf (leaf command, no subcommands)
- **docker**
  - [x] `docker` — ok (43 parsed)
  - [x] `docker compose` — ok (35 parsed)
  - [x] `docker image` — ok (12 parsed)
  - [x] `docker container` — ok (25 parsed)
  - [x] `docker network` — ok (7 parsed)
  - [x] `docker volume` — ok (5 parsed)
  - [x] `docker builder` — ok (16 parsed)
  - [x] `docker context` — ok (9 parsed)
  - [x] `docker plugin` — ok (10 parsed)
- **gcloud**
  - [x] `gcloud` — ok (fixture not committed; parses; fixture not committed (proprietary))
  - [ ] `gcloud config` — bogus (fixture not committed; config properties leak as commands)
- **gh**
  - [x] `gh` — ok (36 parsed)
  - [x] `gh pr` — ok (17 parsed; prose headings rejected)
  - [x] `gh repo` — ok (16 parsed; prose headings rejected)
  - [x] `gh auth` — ok (7 parsed)
  - [x] `gh issue` — ok (15 parsed)
  - [x] `gh run` — ok (7 parsed)
  - [x] `gh run/rerun` — leaf (leaf command, no subcommands)
  - [x] `gh workflow` — ok (5 parsed)
  - [x] `gh release` — ok (8 parsed)
  - [x] `gh extension` — ok (8 parsed)
  - [x] `gh codespace` — ok (13 parsed)
  - [x] `gh cache` — ok (2 parsed)
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
  - [x] `helm dependency` — ok (3 parsed; yaml example blocks skipped)
  - [x] `helm repo` — ok (5 parsed)
  - [x] `helm plugin` — ok (4 parsed)
  - [x] `helm show` — ok (5 parsed)
- [ ] `jq` — out-of-scope (fixture not committed; no subcommand concept)
- **kubectl**
  - [x] `kubectl` — ok (42 parsed)
  - [x] `kubectl config` — ok (15 parsed)
  - [x] `kubectl rollout` — ok (6 parsed)
  - [x] `kubectl api-resources` — leaf (leaf command, no subcommands)
- [ ] `make` — out-of-scope (fixture not committed; no subcommand concept)
- **mise**
  - [x] `mise` — ok (67 parsed)
  - [x] `mise tasks` — ok (9 parsed)
  - [x] `mise plugins` — ok (8 parsed)
- [ ] `mvn` — out-of-scope (goals defined by plugins, not in --help)
- **npm**
  - [x] `npm` — ok (15 parsed; comma inventory format)
  - [x] `npm cache` — ok (4 parsed; 2-token prefixed synopsis lines)
- **pip**
  - [x] `pip` — ok (16 parsed)
  - [x] `pip install` — leaf (leaf command, no subcommands)
- [x] `pnpm` — ok (92 parsed)
- [x] `poetry` — ok (20 parsed; help_args: list)
- [ ] `python3` — out-of-scope (fixture not committed; no subcommand concept)
- **rustup**
  - [x] `rustup` — ok (18 parsed)
  - [x] `rustup toolchain` — ok (5 parsed)
  - [x] `rustup component` — ok (4 parsed)
  - [x] `rustup target` — ok (4 parsed)
- [ ] `ssh` — out-of-scope (fixture not committed; no subcommand concept)
- [ ] `sudo` — out-of-scope (fixture not committed; no subcommand concept)
- [x] `systemctl` — ok (13 parsed; [ARG...] placeholders skipped)
- [x] `tar` — leaf (option-value sections are skipped)
- **terraform**
  - [x] `terraform` — ok (fixture not committed; parses; fixture not committed (BUSL))
  - [x] `terraform state` — ok (fixture not committed; parses; fixture not committed (BUSL))
- [x] `tmux` — ok (85 parsed; col-0 name (alias) entries; help_args: list-commands)
- **uv**
  - [x] `uv` — ok (23 parsed)
  - [x] `uv tool` — ok (8 parsed)
  - [x] `uv python` — ok (8 parsed)
  - [x] `uv pip` — ok (9 parsed)
  - [x] `uv venv` — leaf (leaf command, no subcommands)
- [x] `yarn` — ok (27 parsed; tool-prefixed usage lines)

## Totals

- ok: 73
- leaf: 11
- bogus: 1
- out-of-scope: 8
