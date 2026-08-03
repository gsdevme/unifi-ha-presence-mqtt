# 07 — Deployment

## Container (REQ-DP-01, REQ-DP-02)

A multi-stage Dockerfile:

- **build**: `golang:1.26`, `go mod download` (cached), then a static binary —
  `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/... ./cmd`.
- **runtime**: `gcr.io/distroless/static:nonroot`, copies the single binary,
  `EXPOSE 8080`, runs as `nonroot`.
- Default env `MODE=live`; `ENTRYPOINT ["/unifi-ha-presence-mqtt"]`, `CMD ["serve"]`.

No `time/tzdata` is embedded — there is no timezone-dependent scheduling.

## Module (REQ-DP-03)

Module path `github.com/gsdevme/unifi-ha-presence-mqtt`, `go 1.26`. The UniFi client is
in-tree (`internal/unifi`); the only runtime deps are `paho.golang`, `cobra`,
`godotenv` (plus `godog` for tests).

## CI (REQ-DP-04)

- `.github/workflows/checks.yml` — a reusable `workflow_call` gate running `make lint`,
  `make test`, `make test-e2e` (the Makefile targets are the single source of truth for
  tool versions).
- `.github/workflows/ci.yml` — the PR gate: calls `checks.yml` plus a **build-only**
  amd64 Docker validation (no push) to catch Dockerfile breakage before merge.

## Release (REQ-DP-05)

- `.github/workflows/release.yml` — on push to `master`: run `checks.yml`, then
  `release-please` maintains the Release PR / cuts the tag+Release from Conventional
  Commits. Only when a release is actually created does the gated `image` job build and
  push a multi-arch (`amd64`+`arm64`) image via buildx to
  `ghcr.io/gsdevme/unifi-ha-presence-mqtt:{tag,latest}`, authenticating with the
  built-in `GITHUB_TOKEN`.
- `release-please-config.json` (`{"packages":{".":{"release-type":"go"}}}`) +
  `.release-please-manifest.json` (seed `{".":"0.0.0"}` so the first merge cuts the
  initial semver) + a committed `CHANGELOG.md`.

goreleaser, semantic-release and the DockerHub pipeline are removed.
