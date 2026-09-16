# Downstream Source Lineage

This repository carries a temporary Ross Video patch for ExternalDNS Gateway API route compatibility.

## Upstream baseline

- Repository: <https://github.com/kubernetes-sigs/external-dns>
- Release tag: `v0.22.0`
- Release commit: `994f908d4abdfe5fbf38f2f61613ed570432e43a`
- Linux amd64 image: `registry.k8s.io/external-dns/external-dns:v0.22.0`
- Linux amd64 image digest: `sha256:5fdcaf7deb5c158f93a1fc6fe169cdff4cfb9ae0172bee1a90ab0ef74fb9c9cf`
- License: Apache License 2.0, retained in `LICENSE.md`
- Notice file: the upstream `v0.22.0` tree has no root `NOTICE` file

## Downstream patch

- Patch branch: `ross/v0.22.0-gateway-v1`
- Reserved release tag: `v0.22.0-rv.1`
- Tracking story: [VS-3429](https://rossvideo.atlassian.net/browse/VS-3429)
- Reference change: [external-dns PR #6656](https://github.com/kubernetes-sigs/external-dns/pull/6656)
- Reference commit: `1a1ead3dec04141ad35c1246ebbf49b81fe9d077`
- Reference status: closed without merge, review approval, or successful upstream test evidence

The downstream patch ports only the TCPRoute and UDPRoute API-version behavior from the reference change. It prefers
`gateway.networking.k8s.io/v1`, retains `v1alpha2` as a legacy fallback, and preserves discovery failures instead of
reporting authorization or transport failures as unsupported API versions.

Modified upstream files:

- `docs/sources/gateway-api.md`
- `source/gateway.go`
- `source/gateway_test.go`
- `source/gateway_tcproute.go`
- `source/gateway_tcproute_test.go`
- `source/gateway_udproute.go`
- `source/gateway_udproute_test.go`

## Retirement

Retire this fork after a tagged upstream ExternalDNS release contains equivalent Gateway API v1 TCPRoute and UDPRoute
support and passes the downstream unit, image scan, EKS v1.6.1 Standard CRD, Route53, and MultiViewer acceptance gates.
