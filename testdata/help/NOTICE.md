# Help corpus notices

The fixtures under `testdata/help/` are inputs for the help parser's
corpus test. They are used in three ways:

- `source=capture` fixtures are verbatim help output recorded on a
  developer machine by `cmd/jym-help-capture`. Only CLIs whose licenses
  permit redistribution are committed this way; attribution below.
- `source=synthetic` fixtures are hand-written and merely mimic a help
  format. They are used for CLIs whose licenses (GPL, LGPL, Artistic)
  make committing verbatim output undesirable, and contain no captured
  text beyond short functional command/variable names.
- `source=local` entries have no committed fixture; `jym-help-capture`
  can fill them in on a developer machine for inspection.

Commands are only ever *run* to produce `--help` text; jym contains no
code from these projects.

## Captured fixtures

| Command | Upstream project | License |
| --- | --- | --- |
| docker | Docker CLI (`docker/cli`) | Apache-2.0 |
| gh | GitHub CLI (`cli/cli`) | MIT |
| go | The Go Programming Language | BSD-3-Clause |
| cargo | Cargo (`rust-lang/cargo`) | MIT/Apache-2.0 |
| pip | pip (`pypa/pip`) | MIT |
| aws | AWS CLI (`aws/aws-cli`) | Apache-2.0 |

## Synthetic fixtures (format only, not verbatim)

| Command | Upstream project | License |
| --- | --- | --- |
| git | Git (`git/git`) | GPL-2.0 |
| systemctl | systemd | LGPL-2.1 |
| apt | apt (`Debian/apt`) | GPL-2.0 |
| npm | npm CLI (`npm/cli`) | Artistic-2.0 |
| tar | GNU tar | GPL-3.0 |
