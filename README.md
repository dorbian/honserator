# HonseFarm Operator (with Cloudflared + internal TLS)

This is a minimal, controller-runtime-based operator for the `HonseFarmCluster`
CRD under the API group `clusters.honse.farm`.

It currently does:

* Watches `HonseFarmCluster` resources.
* Ensures the target namespace exists (`spec.namespace`).
* Ensures:
  * a random-key Secret `honsefarm-secrets` in the target namespace;
  * a TLS Secret `honsefarm-tls` with a self-signed certificate covering
    `spec.apiDomain` and any Cloudflared ingress hostnames.
* If `spec.cloudflared.enabled: true`, reconciles:
  * a `cloudflared-config` ConfigMap with `config.yaml`;
  * a `cloudflared` Deployment that runs `cloudflared tunnel run`.

You can extend `controllers/honsefarmcluster_controller.go` to create the
actual HonseFarm server/fileserver/adminpanel Deployments and Services, using
`spec.images` and `spec.registry`.

## Build

```bash
cd operator
go mod tidy
go build ./...
```

Then containerize as usual, e.g. with Podman.

## CRD & manifests

The `manifests/` directory (sibling to `operator/`) contains:

* `crd-honsefarmcluster.yaml` – CRD for `HonseFarmCluster`.
* `sample-honsefarmcluster.yaml` – example instance.
* `cloudflared-deployment-template.yaml` – reference template for Cloudflared.
* `rbac.yaml` – RBAC for the operator.
* `operator-deployment.yaml` – example Deployment for the operator.
* `namespace.yaml` – `honsefarm-system` namespace.
