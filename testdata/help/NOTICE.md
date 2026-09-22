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

| borg | BorgBackup (`borgbackup/borg`) | BSD-3-Clause |
| chezmoi | chezmoi (`twpayne/chezmoi`) | MIT |
| mkdocs | MkDocs (`mkdocs/mkdocs`) | BSD-2-Clause |
| dbt | dbt (`dbt-labs/dbt-core`) | Apache-2.0 |
| podman | Podman (`containers/podman`) | Apache-2.0 |
| hugo | Hugo (`gohugoio/hugo`) | Apache-2.0 |
| rclone | rclone (`rclone/rclone`) | MIT |
| caddy | Caddy (`caddyserver/caddy`) | Apache-2.0 |
| traefik | Traefik (`traefik/traefik`) | MIT |
| restic | restic (`restic/restic`) | BSD-2-Clause |
| velero | Velero (`vmware-tanzu/velero`) | Apache-2.0 |
| promtool | Prometheus promtool (`prometheus/prometheus`) | Apache-2.0 |
| zellij | Zellij (`zellij-org/zellij`) | MIT |
| starship | Starship (`starship/starship`) | ISC |
| task | Task (`go-task/task`) | MIT |
| just | just (`casey/just`) | CC0-1.0 |
| vultr-cli | Vultr CLI (`vultr/vultr-cli`) | Apache-2.0 |
| cosign | cosign (`sigstore/cosign`) | Apache-2.0 |
| trivy | Trivy (`aquasecurity/trivy`) | Apache-2.0 |
| syft | Syft (`anchore/syft`) | Apache-2.0 |
| grype | Grype (`anchore/grype`) | Apache-2.0 |
| tkn | Tekton CLI (`tektoncd/cli`) | Apache-2.0 |
| rancher | Rancher CLI (`rancher/cli`) | Apache-2.0 |
| dagger | Dagger (`dagger/dagger`) | Apache-2.0 |
| goreleaser | GoReleaser (`goreleaser/goreleaser`) | MIT |
| govc | govmomi govc (`vmware/govmomi`) | Apache-2.0 |
| kubectl-krew | krew (`kubernetes-sigs/krew`) | Apache-2.0 |
| certbot | Certbot (`certbot/certbot`) | Apache-2.0 |
| composer | Composer (`composer/composer`) | MIT |
| bundler | Bundler (`rubygems/bundler`) | MIT |
| volta | Volta (`volta-cli/volta`) | BSD-2-Clause |
| alembic | Alembic (`sqlalchemy/alembic`) | MIT |
| ansible-builder | Ansible Builder (`ansible/ansible-builder`) | Apache-2.0 |
| antctl | Antrea (`antrea-io/antrea`) | Apache-2.0 |
| apko | apko (`chainguard-dev/apko`) | Apache-2.0 |
| asdf | asdf (`asdf-vm/asdf`) | MIT |
| atlas | Atlas (`ariga/atlas`) | Apache-2.0 |
| atmos | Atmos (`cloudposse/atmos`) | Apache-2.0 |
| auth0 | Auth0 CLI (`auth0/auth0-cli`) | MIT |
| authelia | Authelia (`authelia/authelia`) | Apache-2.0 |
| autorestic | autorestic (`cupcakearmy/autorestic`) | Apache-2.0 |
| aws-vault | AWS Vault (`99designs/aws-vault`) | MIT |
| azd | Azure Developer CLI (`Azure/azure-dev`) | MIT |
| b2 | B2 CLI (`Backblaze/B2_Command_Line_Tool`) | MIT |
| bicep | Bicep (`Azure/bicep`) | MIT |
| bin | bin (`marcosnils/bin`) | MIT |
| bk | Buildkite CLI (`buildkite/cli`) | MIT |
| bom | bom (`kubernetes-sigs/bom`) | Apache-2.0 |
| buildctl | BuildKit (`moby/buildkit`) | Apache-2.0 |
| calicoctl | Calico (`projectcalico/calico`) | Apache-2.0 |
| cerbosctl | Cerbos (`cerbos/cerbos`) | Apache-2.0 |
| cfssl | CFSSL (`cloudflare/cfssl`) | BSD-2-Clause |
| chalice | Chalice (`aws/chalice`) | Apache-2.0 |
| checkov | Checkov (`bridgecrewio/checkov`) | Apache-2.0 |
| cilium | Cilium (`cilium/cilium`) | Apache-2.0 |
| clusterctl | Cluster API (`kubernetes-sigs/cluster-api`) | Apache-2.0 |
| cmctl | cert-manager cmctl (`cert-manager/cert-manager`) | Apache-2.0 |
| conan | Conan (`conan-io/conan`) | MIT |
| conftest | Conftest (`open-policy-agent/conftest`) | Apache-2.0 |
| copilot | AWS Copilot (`aws/copilot-cli`) | Apache-2.0 |
| crane | crane (`google/go-containerregistry`) | Apache-2.0 |
| crictl | cri-tools (`kubernetes-sigs/cri-tools`) | Apache-2.0 |
| dapr | Dapr CLI (`dapr/cli`) | Apache-2.0 |
| dbmate | dbmate (`amacneil/dbmate`) | MIT |
| ddev | DDEV (`ddev/ddev`) | Apache-2.0 |
| deck | decK (`Kong/deck`) | Apache-2.0 |
| devbox | Devbox (`jetify-com/devbox`) | Apache-2.0 |
| devspace | DevSpace (`devspace-sh/devspace`) | Apache-2.0 |
| dgraph | Dgraph (`dgraph-io/dgraph`) | Apache-2.0 |
| dprint | dprint (`dprint/dprint`) | MIT |
| draft | Draft (`Azure/draft`) | MIT |
| egctl | Envoy Gateway (`envoyproxy/gateway`) | Apache-2.0 |
| etcdctl | etcd (`etcd-io/etcd`) | Apache-2.0 |
| faas-cli | OpenFaaS CLI (`openfaas/faas-cli`) | MIT |
| falcoctl | falcoctl (`falcosecurity/falcoctl`) | Apache-2.0 |
| fga | OpenFGA CLI (`openfga/cli`) | Apache-2.0 |
| func | Knative func (`knative/func`) | Apache-2.0 |
| gator | Gatekeeper (`open-policy-agent/gatekeeper`) | Apache-2.0 |
| ghq | ghq (`x-motemen/ghq`) | MIT |
| git-lfs | Git LFS (`git-lfs/git-lfs`) | MIT |
| git-town | Git Town (`git-town/git-town`) | MIT |
| gitlab-runner | GitLab Runner | MIT |
| goose | goose (`pressly/goose`) | MIT |
| granted | Granted (`common-fate/granted`) | MIT |
| gscloud | gscloud (`gridscale/gscloud`) | MIT |
| gsutil | gsutil (`GoogleCloudPlatform/gsutil`) | Apache-2.0 |
| gum | gum (`charmbracelet/gum`) | MIT |
| hub | hub (`github/hub`) | MIT |
| hubble | Hubble (`cilium/hubble`) | Apache-2.0 |
| hydra | Ory Hydra (`ory/hydra`) | Apache-2.0 |
| ignite | Ignite (`weaveworks/ignite`) | Apache-2.0 |
| infracost | Infracost (`infracost/infracost`) | Apache-2.0 |
| ionosctl | IONOS Cloud CLI (`ionos-cloud/ionosctl`) | Apache-2.0 |
| jx | Jenkins X (`jenkins-x/jx`) | Apache-2.0 |
| k0s | k0s (`k0sproject/k0s`) | Apache-2.0 |
| k3d | k3d (`k3d-io/k3d`) | MIT |
| k8sgpt | K8sGPT (`k8sgpt-ai/k8sgpt`) | Apache-2.0 |
| keptn | Keptn (`keptn/keptn`) | Apache-2.0 |
| keto | Ory Keto (`ory/keto`) | Apache-2.0 |
| kn | Knative CLI (`knative/client`) | Apache-2.0 |
| ko | ko (`ko-build/ko`) | Apache-2.0 |
| kopia | Kopia (`kopia/kopia`) | Apache-2.0 |
| kops | kOps (`kubernetes/kops`) | Apache-2.0 |
| kratos | Ory Kratos (`ory/kratos`) | Apache-2.0 |
| kube-bench | kube-bench (`aquasecurity/kube-bench`) | Apache-2.0 |
| kube-capacity | kube-capacity (`robscott/kube-capacity`) | Apache-2.0 |
| kube-score | kube-score (`zegl/kube-score`) | MIT |
| kubeaudit | kubeaudit (`Shopify/kubeaudit`) | MIT |
| kubecm | kubecm (`sunny0826/kubecm`) | Apache-2.0 |
| kubectl-ai | kubectl-ai (`GoogleCloudPlatform/kubectl-ai`) | Apache-2.0 |
| kubectl-argo-rollouts | Argo Rollouts (`argoproj/argo-rollouts`) | Apache-2.0 |
| kubectl-who-can | kubectl-who-can (`aquasecurity/kubectl-who-can`) | Apache-2.0 |
| kubectx | kubectx (`ahmetb/kubectx`) | Apache-2.0 |
| kubefwd | kubefwd (`txn2/kubefwd`) | Apache-2.0 |
| kubeone | KubeOne (`kubermatic/kubeone`) | Apache-2.0 |
| kubescape | Kubescape (`kubescape/kubescape`) | Apache-2.0 |
| kubeseal | kubeseal (`bitnami-labs/sealed-secrets`) | Apache-2.0 |
| kyverno | Kyverno (`kyverno/kyverno`) | Apache-2.0 |
| lab | lab (`zaquestion/lab`) | CC0-1.0 |
| lego | lego (`go-acme/lego`) | MIT |
| litecli | litecli (`dbcli/litecli`) | BSD-3-Clause |
| litefs | LiteFS (`superfly/litefs`) | Apache-2.0 |
| melange | melange (`chainguard-dev/melange`) | Apache-2.0 |
| micromamba | micromamba (`mamba-org/mamba`) | BSD-3-Clause |
| migrate | migrate (`golang-migrate/migrate`) | MIT |
| mirrord | mirrord (`metalbear-co/mirrord`) | MIT |
| molecule | Molecule (`ansible/molecule`) | MIT |
| mongosh | MongoDB Shell (`mongodb-js/mongosh`) | Apache-2.0 |
| mycli | mycli (`dbcli/mycli`) | BSD-3-Clause |
| nats | NATS CLI (`nats-io/natscli`) | Apache-2.0 |
| nerdctl | nerdctl (`containerd/nerdctl`) | Apache-2.0 |
| nfpm | nFPM (`goreleaser/nfpm`) | MIT |
| nitric | Nitric CLI (`nitrictech/cli`) | Apache-2.0 |
| notation | Notation (`notaryproject/notation`) | Apache-2.0 |
| nsc | nsc (`nats-io/nsc`) | Apache-2.0 |
| nuctl | Nuclio (`nuclio/nuclio`) | Apache-2.0 |
| oathkeeper | Ory Oathkeeper (`ory/oathkeeper`) | Apache-2.0 |
| oci | OCI CLI (`oracle/oci-cli`) | UPL-1.0/Apache-2.0 |
| ocm | OCM (`open-component-model/ocm`) | Apache-2.0 |
| okteto | Okteto (`okteto/okteto`) | Apache-2.0 |
| opa | OPA (`open-policy-agent/opa`) | Apache-2.0 |
| openstack | OpenStackClient (`openstack/python-openstackclient`) | Apache-2.0 |
| oras | ORAS (`oras-project/oras`) | Apache-2.0 |
| osv-scanner | OSV-Scanner (`google/osv-scanner`) | Apache-2.0 |
| pipenv | Pipenv (`pypa/pipenv`) | MIT |
| pixi | pixi (`prefix-dev/pixi`) | BSD-3-Clause |
| pocketbase | PocketBase (`pocketbase/pocketbase`) | MIT |
| popeye | Popeye (`derailed/popeye`) | Apache-2.0 |
| porter | Porter (`getporter/porter`) | Apache-2.0 |
| pre-commit | pre-commit (`pre-commit/pre-commit`) | MIT |
| pscale | PlanetScale CLI (`planetscale/cli`) | Apache-2.0 |
| pyenv | pyenv (`pyenv/pyenv`) | MIT |
| rbac-tool | rbac-tool (`alcideio/rbac-tool`) | Apache-2.0 |
| rekor-cli | Rekor (`sigstore/rekor`) | Apache-2.0 |
| rke | RKE (`rancher/rke`) | Apache-2.0 |
| rosa | ROSA CLI (`openshift/rosa`) | Apache-2.0 |
| rqlite | rqlite (`rqlite/rqlite`) | MIT |
| ruff | Ruff (`astral-sh/ruff`) | MIT |
| runsc | gVisor runsc (`google/gvisor`) | Apache-2.0 |
| rustic | rustic (`rustic-rs/rustic`) | Apache-2.0 |
| saml2aws | saml2aws (`Versent/saml2aws`) | MIT |
| scw | Scaleway CLI (`scaleway/scaleway-cli`) | Apache-2.0 |
| spacectl | spacectl (`spacelift-io/spacectl`) | MIT |
| sqlc | sqlc (`sqlc-dev/sqlc`) | MIT |
| stackit | STACKIT CLI (`stackitcloud/stackit-cli`) | Apache-2.0 |
| step | step CLI (`smallstep/cli`) | Apache-2.0 |
| stern | stern (`stern/stern`) | Apache-2.0 |
| terragrunt | Terragrunt (`gruntwork-io/terragrunt`) | MIT |
| terrascan | Terrascan (`tenable/terrascan`) | Apache-2.0 |
| topaz | Topaz (`aserto-dev/topaz`) | MIT |
| tox | tox (`tox-dev/tox`) | MIT |
| turso | Turso CLI (`tursodatabase/turso-cli`) | MIT |
| twine | Twine (`pypa/twine`) | Apache-2.0 |
| ubi | ubi (`houseabsolute/ubi`) | Apache-2.0 |
| upctl | UpCloud CLI (`UpCloudLtd/upcloud-cli`) | MIT |
| vcluster | vCluster (`loft-sh/vcluster`) | Apache-2.0 |
| woodpecker-cli | Woodpecker CI (`woodpecker-ci/woodpecker`) | Apache-2.0 |
| wp | WP-CLI (`wp-cli/wp-cli`) | MIT |
| zed | zed (`authzed/zed`) | Apache-2.0 |

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
| zola | Zola (`getzola/zola`) | EUPL-1.2 |
| symfony | Symfony CLI (`symfony-cli/symfony-cli`) | AGPL-3.0 |
| zitadel | ZITADEL (`zitadel/zitadel`) | AGPL-3.0 |
| garden | Garden (`garden-io/garden`) | MPL-2.0 |
| terramate | Terramate (`terramate-io/terramate`) | MPL-2.0 |
