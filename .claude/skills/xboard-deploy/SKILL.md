---
name: xboard-deploy
description: Docker/Octane deployment topologies and local dev setup for Xboard (Caddy + PHP + Supervisor container, compose variants, install flow). Use when changing Dockerfile/.docker configs, docker-compose files, environment variables, or debugging a deployment/Octane/queue issue.
---

# Xboard — deployment

## Container shape

Single `Dockerfile` builds an all-in-one image containing **Caddy (web server) + PHP + Supervisor**,
configured under `.docker/{caddy,entrypoint.sh,php,supervisor}`. Supervisor is what keeps
**Octane** (the app server, running on Workerman) and **Horizon** (queue worker dashboard)
running as long-lived processes inside the container — this is not a stateless php-fpm request
model, so code changes to singletons/statics need a process restart (not just a page reload) to
take effect in a running container.

## Compose variants (pick based on topology, don't assume one is canonical)

- `compose.sample.yaml` — standalone all-in-one.
- `compose.host.sample.yaml` — host-network style deployment.
- `compose.split.sample.yaml` — services split apart (e.g. DB/Redis external or separate
  containers) rather than bundled.
- `compose.1panel.sample.yaml` — tailored for 1Panel-managed hosts.

Copy the relevant sample to `compose.yaml` (or `docker-compose.yml`, per that file's own
instructions) rather than editing the sample in place, so local overrides don't get committed.

## First-run install

`.env.example` → `.env`, set `INSTALLED=false` initially; the `xboard:install` artisan command
(triggered via `init.sh` or manually) runs the setup wizard and flips `INSTALLED=true`.
`update.sh` is the upgrade path for existing installs — prefer it over ad hoc `artisan migrate`
when upgrading a running deployment, since it likely also handles cache-clearing/asset steps.

## Required external services

MySQL (primary DB), Redis (required for both cache and queue — Octane/Horizon depend on it, not
optional). Mail via SMTP or Mailgun. Optional: Google Cloud Storage (`cloud_storage.php` config),
Google reCAPTCHA, Stripe (payments).

## Docs

`docs/` includes deployment guides for 1Panel, Docker Compose, and aaPanel, plus migration
guides from v2board — check there before inventing a new deployment path.

## Related

See [[xboard-architecture]] for the app-level structure (Octane/Workerman implications on state,
API routing, models).
