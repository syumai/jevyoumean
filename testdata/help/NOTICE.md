# Help corpus notices

The fixtures under `testdata/help/` are inputs for the help parser's
corpus test. They are used in three ways:

- `source=capture` fixtures are verbatim help output recorded on a
  developer machine by `cmd/jym-help-capture`. Only CLIs whose licenses
  permit redistribution are committed this way; attribution below.
- `source=synthetic` fixtures are hand-written and merely mimic a help
  format. They are used for CLIs whose licenses (GPL, LGPL, Artistic,
  MPL and other copyleft licenses) make committing verbatim output
  undesirable, and contain no captured text beyond short functional
  command/variable names.
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
| kubectl | kubectl (`kubernetes/kubernetes`) | Apache-2.0 |
| helm | Helm (`helm/helm`) | Apache-2.0 |
| mise | mise (`jdx/mise`) | MIT |
| uv | uv (`astral-sh/uv`) | MIT/Apache-2.0 |
| deno | Deno (`denoland/deno`) | MIT |
| bun | Bun (`oven-sh/bun`) | MIT |
| rustup | rustup (`rust-lang/rustup`) | MIT/Apache-2.0 |
| tmux | tmux (`tmux/tmux`) | ISC |
| pnpm | pnpm (`pnpm/pnpm`) | MIT |
| poetry | Poetry (`python-poetry/poetry`) | MIT |
| yarn | Yarn (`yarnpkg/berry`) | BSD-2-Clause |
| az | Azure CLI (`Azure/azure-cli`) | MIT |
| brew | Homebrew (`Homebrew/brew`) | BSD-2-Clause |
| mvn | Apache Maven | Apache-2.0 |
| doctl | DigitalOcean CLI (`digitalocean/doctl`) | Apache-2.0 |
| flyctl | flyctl (`superfly/flyctl`) | Apache-2.0 |
| stripe | Stripe CLI (`stripe/stripe-cli`) | Apache-2.0 |
| glab | GitLab CLI (`gitlab-org/cli`) | MIT |
| hcloud | Hetzner Cloud CLI (`hetznercloud/cli`) | MIT |
| eksctl | eksctl (`eksctl-io/eksctl`) | Apache-2.0 |
| kind | kind (`kubernetes-sigs/kind`) | Apache-2.0 |
| k9s | k9s (`derailed/k9s`) | Apache-2.0 |
| minikube | minikube (`kubernetes/minikube`) | Apache-2.0 |
| kustomize | kustomize (`kubernetes-sigs/kustomize`) | Apache-2.0 |
| helmfile | helmfile (`helmfile/helmfile`) | MIT |
| argocd | Argo CD CLI (`argoproj/argo-cd`) | Apache-2.0 |
| tilt | Tilt (`tilt-dev/tilt`) | MIT |
| skaffold | Skaffold (`GoogleContainerTools/skaffold`) | Apache-2.0 |
| pulumi | Pulumi CLI (`pulumi/pulumi`) | Apache-2.0 |
| linkerd | Linkerd CLI (`linkerd/linkerd2`) | Apache-2.0 |
| flux | Flux CLI (`fluxcd/flux2`) | Apache-2.0 |
| vercel | Vercel CLI (`vercel/vercel`) | Apache-2.0 |
| netlify | Netlify CLI (`netlify/cli`) | MIT |
| railway | Railway CLI (`railwayapp/cli`) | MIT |
| wrangler | Wrangler (`cloudflare/workers-sdk`) | MIT/Apache-2.0 |
| pipx | pipx (`pypa/pipx`) | MIT |
| hatch | Hatch (`pypa/hatch`) | MIT |
| pdm | PDM (`pdm-project/pdm`) | MIT |
| meson | Meson (`mesonbuild/meson`) | Apache-2.0 |
| rye | Rye (`astral-sh/rye`) | MIT |
| istioctl | Istio (`istio/istio`) | Apache-2.0 |
| pack | pack (`buildpacks/pack`) | Apache-2.0 |
| supabase | Supabase CLI (`supabase/cli`) | MIT |
| firebase | Firebase CLI (`firebase/firebase-tools`) | MIT |

## Synthetic fixtures (format only, not verbatim)

| Command | Upstream project | License |
| --- | --- | --- |
| git | Git (`git/git`) | GPL-2.0 |
| systemctl | systemd | LGPL-2.1 |
| apt | apt (`Debian/apt`) | GPL-2.0 |
| npm | npm CLI (`npm/cli`) | Artistic-2.0 |
| tar | GNU tar | GPL-3.0 |
| sops | SOPS (`getsops/sops`) | MPL-2.0 |
| tofu | OpenTofu (`opentofu/opentofu`) | MPL-2.0 |
