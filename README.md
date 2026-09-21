# jevyoumean / `jym`

`jym` wraps any CLI command. When you enter a subcommand the CLI does not
document, it reads the CLI's `--help` output and asks
[Jev](https://typesafe.ai) — TypeSafe's System One model — for a semantic
`Did you mean?` suggestion.

```console
$ jym -- gh pr vie

Unknown subcommand "vie".

Did you mean "view"?
```

Unlike a conventional edit-distance `Did you mean?`, `jym` matches on
*intent*: it hands Jev the candidate subcommand names plus their help
descriptions. `show` and `view` share no characters worth mentioning, but
Jev knows they mean the same thing.

This is a small experiment: can semantic distance replace edit distance
for CLI "Did you mean?" suggestions?

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

The `--` separator is recommended but optional (`jym git switch main`
also works). Use `--` whenever the target command's name collides with
jym's own subcommands (`auth`, `help`, `version`).

The intended use is as a project-local shell alias, e.g. with
[mise](https://mise.jdx.dev):

```toml
[shell_alias]
gh = "jym -- gh"
kubectl = "jym -- kubectl"
```

`jym` resolves the target with `exec.LookPath`, so the alias never
recursively expands.

## First-run setup

On first run without a configured key, `jym` asks for a TypeSafe API key
(input is not echoed):

```console
$ jym -- gh pr view 123

jym needs a TypeSafe API key to use Jev.

TypeSafe API key: **************

✓ API key saved.
```

The key is looked up in this order:

1. `TYPESAFE_API_KEY` environment variable
2. `jym`'s local configuration file

```console
$ jym auth status    # API key: configured
$ jym auth set       # prompt and store
$ jym auth remove    # delete the stored key
```

## How it works

Jev is only called when needed — a valid command never triggers a
network request:

1. `jym -- gh pr view 123` → resolve `gh` → `gh --help` says `pr` exists
   → `gh pr --help` says `view` exists → `exec gh pr view 123`. No
   network access.
2. `jym -- gh pur request` → `pur` is not documented → Jev `Choice` over
   the documented candidates (plus a mandatory `__none__` escape hatch)
   → `Did you mean "repo"?` if the answer clears the thresholds.

`jym` only recommends; it never auto-corrects. The original command is
always executed unchanged, so the target CLI still prints its own error
and exit code.

## Subcommand detection

The first non-option argument is treated as the subcommand, recursively
(depth limit: 3). Heuristics apply: `--flag=value` is self-contained, a
bare flag makes the next argument ambiguous (it may be a flag value), and
`--` ends option parsing. When in doubt, `jym` keeps quiet — false
negatives are preferred over false positives.

Help parsing is heuristic: sections like `Commands:`,
`Available Commands:`, `CORE COMMANDS`, `Subcommands:` are recognized,
with both `name   description` and `name:   description` entry formats.

## Cache

Parsed help output is cached under `~/.cache/jevyoumean/`
(`$XDG_CACHE_HOME/jevyoumean`), keyed by executable path and mtime. A
reinstalled or upgraded CLI automatically invalidates the cache.

## Configuration

`$XDG_CONFIG_HOME/jevyoumean/config.toml` (or
`~/.config/jevyoumean/config.toml` on macOS), file mode `0600`:

```toml
api_key = "..."
min_probability = 0.60
min_confidence  = 0.50
timeout         = "1s"
debug           = false
```

| Setting           | Default | Meaning                                        |
| ----------------- | ------- | ---------------------------------------------- |
| `min_probability` | `0.60`  | Minimum probability of Jev's top choice        |
| `min_confidence`  | `0.50`  | Minimum overall confidence of the answer       |
| `timeout`         | `1s`    | Client-side timeout for the TypeSafe API       |

If `__none__` wins or the thresholds are not met, no suggestion is shown.
If the API call fails or times out, the suggestion is silently skipped —
a broken `jym` must never break the wrapped command.

## Debug mode

```console
$ JYM_DEBUG=1 jym -- gh pr show 123
# or
$ jym --debug -- gh pr show 123
```

```text
[jym] executable: /opt/homebrew/bin/gh
[jym] command path: gh -> pr
[jym] unknown subcommand: show
[jym] candidates:
[jym]   checkout     Check out a pull request
[jym]   view         View a pull request
[jym] jev:
[jym]   __none__     0.01
[jym]   view         0.91
[jym] confidence: 0.86
[jym] latency: 112ms
```

`JYM_API_ENDPOINT` overrides the API endpoint (useful for tests).

## Scope

Implemented: single Go binary, `jym -- command`, first-run API key setup,
`TYPESAFE_API_KEY`, TypeSafe HTTP API + Jev `Choice`, `--help`
introspection, common help formats, nested subcommand detection, cache,
configurable thresholds, `__none__`, debug output, stdio and exit-code
passthrough.

Deliberately not implemented: shell completion, auto-correction, full CLI
grammar parsing, option/argument typo correction, natural-language
command generation, OS keychain integration.

## License

MIT
