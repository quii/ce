---
id: 0023
title: Disable BuildKit for local docker compose builds
status: Accepted
scope: []
enforcement: process
---

# 0023: Disable BuildKit for local docker compose builds

## Decision

The `Up` mage target sets `DOCKER_BUILDKIT=0` when invoking `docker compose up --build`.

## Rationale

On machines with Zscaler (or similar TLS-inspecting proxies), BuildKit's isolated network sandbox does not inherit the host's injected CA certificate, so `go mod download` fails with an x509 verification error. The legacy builder runs inside the Docker daemon directly and picks up OrbStack's CA injection at the VM level, so it succeeds.

Testcontainers uses the legacy `ImageBuild` Docker API by default, which is why the container acceptance tests pass on the same machine where `docker compose up --build` fails.

## Consequences

BuildKit-only features (cache mounts, `--secret`, parallel stages) are unavailable in local compose builds. None are currently used.
