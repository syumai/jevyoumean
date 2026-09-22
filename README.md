# jevyoumean / `jym`

`jym` wraps any CLI command. When you enter a subcommand the CLI does not
document, it reads the CLI's help output and asks
[Jev](https://typesafe.ai) — TypeSafe's System One model — which
documented subcommand you most likely meant.

![jym correcting npm and git subcommands](demo/readme.gif)

```console
$ git remove foo.txt          # actually an alias for: jym git remove foo.txt
jym: "remove" is not a git subcommand. Did you mean?
  1) git rm foo.txt   0.93
  [Enter] run 1   [o] run as typed   [n] cancel
```

Unlike an edit-distance `Did you mean?`, `jym` matches on *intent*: it
hands Jev the candidate subcommand names plus their help descriptions.
`remove → rm`, `list → ps`, `undo → restore` are close in meaning but far
in spelling — that is the gap this experiment targets.

## Install

```console
$ go install github.com/syumai/jevyoumean/cmd/jym@latest
```

## Usage

```console
$ jym -- git switch main
$ jym -- gh pr view 123
$ jym -- kubectl delete pod foo
```

The `--` separator is optional but recommended. `jym` itself has no
subcommands — management operations are flags, so wrapping a command
named `auth` or `setup` never collides.

The intended use is as a project-local shell alias, e.g. with
[mise](https://mise.jdx.dev):

```toml
# mise.toml
[shell_alias]
git = "jym -- git"
gh  = "jym -- gh"
kubectl = "jym -- kubectl"
```

`jym --print-mise git gh kubectl` emits that snippet. `jym` resolves the
target with a PATH lookup, so the alias never recursively expands (a shim
or symlink to `jym` is skipped, and `JYM_DEPTH` breaks any residual
loop).

mise shell aliases only apply in `mise activate`d interactive shells —
which is exactly where `jym` intervenes anyway.

### Bash and zsh integration

To wrap selected commands without mise, add one of these lines to
`.bashrc` or `.zshrc` respectively:

```sh
eval "$(jym --shell-integration bash git gh kubectl)"
eval "$(jym --shell-integration zsh git gh kubectl)"
```

The generated shell functions preserve argument boundaries, redirections
and exit codes. An explicitly selected command replaces an alias with the
same name. Use `command git ...` to bypass a wrapper temporarily.

Experimental and dangerous: `--all` wraps every external executable
currently visible on `PATH`:

```sh
eval "$(jym --shell-integration zsh --all)"
```

Shell builtins, keywords, existing aliases/functions, and `jym` itself are
not wrapped. PATH changes after shell startup are not picked up until the
integration is evaluated again. This mode can misinterpret ordinary
arguments as subcommands (for example, a filename passed to `rm` or
`bash`), add startup overhead, and cause prompts in many commands. It is
not recommended as a default setup.

## Flags

| Flag                       | Action                                                        |
| -------------------------- | ------------------------------------------------------------- |
| `--setup`                  | Prompt for and store the TypeSafe API key (verified, hidden)  |
| `--explain`                | Show extraction, Jev request/response and decision; no exec   |
| `--refresh`                | Discard the wrapped command's help cache and refetch          |
| `--cache-clear`            | Remove the whole help cache                                   |
| `--print-mise <cmd>...`    | Print a `[shell_alias]` snippet for mise.toml                 |
| `--completion <shell>`     | Print a delegating completion script (bash, zsh, fish)        |
| `--shell-integration <shell> <cmd>...` | Wrap selected commands in bash or zsh            |
| `--shell-integration <shell> --all` | **Dangerous:** wrap all external commands on PATH |
| `--doctor`                 | Diagnose key, API reachability, cache and TTY state           |
| `--debug`                  | Debug output on stderr (also `JYM_DEBUG=1`)                   |
| `--version`, `--help`      |                                                               |

## How it works

Check-first, never run-first: `jym` inspects the argument vector against
the documented subcommand tree *before* executing, so side effects can
never run twice.

1. **Gate** — if stderr is not a TTY (scripts, CI, pipes), or `JYM_DEPTH`
   shows jym inside jym, the command executes untouched.
2. **Detect** — the first non-flag argument is the subcommand candidate.
   If flags precede it (the token may be a flag value), jym passes
   through. A match recurses into `<cmd> <sub> --help` (depth limit:
   `max_depth`, default 2). Tokens confirmed before ("learned"),
   configured `extra_subcommands`, and `<cmd>-<token>` plugin
   executables count as valid.
3. **Ask** — only for an unknown token, `jym` sends one Jev `Choice`
   question whose criteria are the candidate names + help descriptions +
   a mandatory `__none__` escape hatch. More than 254 candidates shard
   into a two-phase choice.
4. **Decide** — by mode:
   - `prompt` (default): list up to 3 candidates ≥ `suggest_threshold`,
     wait for one key — `Enter`/`1`-`3` run the corrected command, `o`
     runs as typed, `n`/`Esc`/`Ctrl-C` cancel (exit 127).
   - `hint`: print the list, run as typed.
   - `auto`: run the correction only when p ≥ `auto_run_threshold` and
     the winner is not denylisted (`rm`, `delete`, `destroy`, `reset`,
     `push`, ...); otherwise fall back to prompt.
   - Without a TTY stdin, `prompt`/`auto` degrade to `hint`.
5. **Execute** — on Unix via `syscall.Exec` (native signals, TTY, exit
   codes, job control); elsewhere via a child process with the exit code
   propagated.

Jev is only called on the unknown-subcommand path — valid commands never
touch the network. The hot path is one cached JSON read, well under 5ms.

## Offline fallback

Without an API key or on API failure, `jym` falls back to edit-distance
matching (edit distance ≤ 2 or ≤ len/3, plus prefix matches),
shown without probabilities and marked `(offline)`. When Jev answers —
even `__none__` — its verdict stands and no fallback runs.

"Run as typed" on a command that then exits 0 records the token as
`learned` in the cache, so undocumented-but-valid subcommands (private
aliases, plugins) stop prompting.

## First-run setup

On the first interactive run without a key, `jym` offers setup once
(input hidden, verified with a minimal API call, Enter to skip).
Skipping is recorded in `$XDG_STATE_HOME/jym/state.json` and never
re-asked; `jym --setup` re-runs it anytime.

Key resolution order:

1. `TYPESAFE_API_KEY` environment variable
2. `$XDG_CONFIG_HOME/jym/credentials.toml` (mode `0600`)

## Configuration

`$XDG_CONFIG_HOME/jym/config.toml` (or `~/.config/jym/config.toml`;
`JYM_CONFIG` overrides the path so mise `[env]` can switch per project):

```toml
mode = "prompt"              # prompt | hint | auto
suggest_threshold = 0.30
auto_run_threshold = 0.95
min_confidence    = 0.50     # Jev answer confidence gate
timeout_ms        = 1500
max_depth         = 2        # nested subcommand inspection depth
context_args      = "none"   # none | flags | all — args sent to the API
model             = "jev-latest"
debug             = false
denylist          = []       # additional subcommands never auto-run

[commands.git]
help_args = ["help", "-a"]         # git --help omits most subcommands
extra_subcommands = ["co", "br"]   # your aliases, never prompted on

[commands.kubectl]
max_depth = 3
```

Environment overrides: `JYM_MODE`, `JYM_DEBUG`, `JYM_COLOR` (`always` or
`never`; TTY detection by default), and `JYM_API_ENDPOINT` (endpoint override,
for tests). The standard `NO_COLOR` variable disables colored output.

## Cache

`$XDG_CACHE_HOME/jym/<command>/<key>.json`, keyed by resolved executable
path + mtime + size + subcommand path + `help_args`. Entries carry
`fetched_at` (7-day TTL), `is_leaf`, `subcommands` and `learned`.
A rebuilt binary invalidates automatically; corruption is ignored —
the cache is fail-open.

## Privacy

Sent to the TypeSafe API: the command name, subcommand path, the mistyped
token, and the candidate subcommand names + descriptions. Arguments are
not sent by default (`context_args = "none"`); `flags` sends flag names
only, `all` sends everything. Note that wrapping an internal CLI sends
its subcommand structure to TypeSafe.

## Evaluation

`jym-eval` measures the experiment — semantic vs. edit-distance matching:

```console
$ cat evals.tsv
gh	vie	view
kubectl	del	delete
git	banana	            # empty expected = should NOT suggest
$ jym-eval evals.tsv
```

Each row is `command <TAB> typed <TAB> expected`. The report shows
correct/false-suggestion/miss counts for Jev and for the fallback, plus
latency percentiles.

## Scope

Not implemented (by design): flag/argument correction, command-name
correction (`gti`→`git`), intervention in non-interactive environments,
natural-language command generation, OS keychain integration.

## License

MIT
