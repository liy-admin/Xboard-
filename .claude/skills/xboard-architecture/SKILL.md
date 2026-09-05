---
name: xboard-architecture
description: Code architecture for Xboard, the Laravel-based proxy subscription panel (V2Ray/Shadowsocks/Trojan) — versioned API routing, models, and the split admin/user frontend subprojects. Use when writing or modifying PHP code in app/, adding an API endpoint, touching a model/migration, or working on the admin (React) or user (Vue) frontend.
---

# Xboard — architecture

Laravel 12 (PHP 8.2+) backend, run under **Octane + Workerman** (not plain php-fpm request
lifecycle — be careful with static/singleton state, it persists across requests). Successor to
"v2board"; manages proxy-protocol subscriptions (V2Ray/Shadowsocks/Trojan) that clients like
Hiddify connect to and consume as subscription URLs.

## Backend layout (`app/`)

`Console, Contracts, Exceptions, Helpers, Http, Jobs, Models, Observers, Protocols, Providers,
Scope, Services, Support, Traits, Utils, WebSocket`.

- **API routing is NOT in `routes/api.php`** (that file doesn't exist here). It's versioned and
  lives under `app/Http/Routes/{V1,V2}`, with matching controllers split by actor:
  - V1: `Controllers/V1/{Client,Guest,Passport,Server,User}`
  - V2: `Controllers/V2/{Admin,Client,Server}`
  When adding an endpoint, add it to the correct version+actor pair rather than a generic
  `routes/web.php` entry. `routes/web.php` itself only has `channels.php`/`console.php`/`web.php`
  at top level — no loose API routes there.
- `app/Protocols/` — protocol-specific logic for V2Ray/Shadowsocks/Trojan subscription
  generation; look here first when a new protocol or subscription format is requested.
- `app/Models/` (30 files) is the domain map: `User`, `Plan`, `Order`, `Payment`, `Server`,
  `ServerGroup`, `ServerMachine`, `ServerRoute`, `ServerLog`, `Coupon`, `GiftCardTemplate/Code/
  Usage`, `InviteCode`, `Ticket`/`TicketMessage`, `Notice`, `MailLog`/`MailTemplate`, `Setting`,
  `Plugin`, `CommissionLog`, `ReferralDurationLog`, `TrafficResetLog`, `Stat`/`StatServer`/
  `StatUser`/`ServerStat`, `LuckyDrawLog`, `AdminAuditLog`, `Knowledge`, `SubscribeTemplate`.
  This is a full subscription SaaS: billing/orders/payments, referral/commission, tickets,
  multi-node server management, traffic stats/resets, gift cards, coupons.
- `database/migrations` (47 files) + `database/factories`/`seeders` — standard Laravel migration
  workflow (`php artisan migrate`).
- `plugins/` and `plugins-core/` — Xboard has a plugin system (see `docs/` for the plugin
  development guide) — prefer building optional/extension features as a plugin rather than
  patching core `app/` code, if the feature is genuinely optional.

## Frontends (NOT under `resources/js` — that dir is a near-empty Laravel default)

- **Admin panel**: `admin-src/source/` — separate React 18 + Vite 5 project (Shadcn UI), own
  `package.json`, with `router/`, `store/`, `views/`, `components/`, `composables/`, `api/`,
  `i18n/`, `locales/`. Build/dev this independently of the Laravel app (`cd admin-src/source`).
- **User-facing frontend**: pulled in as a **git submodule** under `theme/Xboard` — Vue 3 +
  TypeScript + NaiveUI, per README. Lives in its own repo; don't expect its source inline here
  unless the submodule is checked out (`git submodule update --init --recursive`).
- Root `package.json` is trivial (just `chokidar`) — not where real frontend deps live.

## Config

`config/` has standard Laravel config plus Xboard-specific: `cloud_storage.php`,
`hidden_features.php`, `theme`, `scribe.php` (generates API docs — regenerate after route
changes if docs matter). `.env.example` covers MySQL, Redis (cache/queue), SMTP/Mailgun,
Google Cloud Storage. `INSTALLED=false` flag gates the `xboard:install` artisan command
(first-run setup wizard).

## Testing

PHPUnit 11. Current coverage is narrow: `tests/Feature/Server/ServerHandshakeTest.php`,
`tests/Unit/Services/Auth/{LoginServiceTest,RegisterServiceTest}.php`. Run with
`php artisan test` or `vendor/bin/phpunit`. Static analysis via Larastan/PHPStan
(`phpstan.neon` at root).

## Related

See [[xboard-deploy]] for Docker/Octane/deployment specifics.
