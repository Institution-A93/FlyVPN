# FLY VPN — Backend Requirements: User Database & Orchestrator

**Audience:** backend specialist
**Status:** draft v3 — all §13 open items resolved by product. Built against the real repo (`Institution-A93/FlyVPN`, the MMVP archive). Key MVP shift vs MMVP: **payments move to platega.io**, **stable VPN credentials**, **one-shot traffic bucket**, new **account-api** service.
**Scope:** the **user database** and the **orchestrator** (control-plane logic) needed to put the **website** (the FlyVPN web prototype) on top of the existing MMVP VPN backend.
**Out of scope:** the VPN data plane internals (sing-box/strongSwan/RADIUS — already built), the Telegram bot's own logic, the web frontend. Their contracts with the orchestrator *are* in scope.

> Read first: `MMVP-REPORT.md`, `README.md` §4 (data model), `docs/adr/0005,0013,0014,0015,0018`, `services/db/migrations/0001_init.up.sql`, `services/config-api/`, `services/orchestrator/`.
> The website's client-facing contract is mocked in the web repo at `src/lib/types.ts` / `src/lib/mockApi.ts`.

---

## ⚠️ Пересмотр скоупа (ADR-0021) — заменяет план ниже

> **Этот документ (draft v3) описывает самописную Go-плоскость entitlement. ADR-0021
> переводит её на FreeRADIUS-native (PostgreSQL); billing = один тонкий webhook.
> Разделы ниже сохранены как история и контракт периферии, но в части entitlement
> читать через призму ADR-0021.**

**Задача тривиальна:** покупка → доступ на срок + кап трафика. Поэтому плоскость
доступа берём нативной (FreeRADIUS + PostgreSQL, спайк-проверено), без платформ
поверх (OpenWISP/SHM/BSS — оверхед, см. `docs/research/billing-and-bss-options.md`).

**Минимальная форма:**

- **Источник истины** — каноническая FreeRADIUS-схема на PostgreSQL 16
  (`radcheck`/`radacct`/`radusergroup`); EAP-MSCHAPv2 из `NT-Password` через `rlm_sql`
  (ADR-0014). Квота — `rlm_sqlcounter`; отрезание — CoA/Disconnect (strongSwan `dae`).
- **Billing** — один тонкий webhook: Platega `CONFIRMED` → upsert `radcheck`
  (срок + кап) + сброс счётчика. **One-shot**, без лицевого счёта/периодики.
- **Выдача** — `.mobileconfig` (`config-api`). **Эксплуатация** — `psql` + SQL-вью;
  GUI-панель (OpenWISP) опциональна и отложена.
- **Telegram-вход** — координация с параллельным треком бота.

**Растворяется из ранее описанного/написанного:**

| Было (draft v3) | Становится (ADR-0021) |
|-----------------|------------------------|
| `usage_log`, bespoke `subscriptions`/quota | каноническая FR-схема + `rlm_sqlcounter` |
| Оркестратор: cron расхода/сброса/порогов/реф-расчёта | нативный счётчик + CoA; у оркестратора остаётся узлы/health/ротация |
| `account-api`: auth/сессии/entitlements/devices | снять; доступ — нативный FreeRADIUS |
| Миграции `0003`–`0005` | superseded |
| `account-api` целиком | тонкий webhook Platega → `radcheck` + `.mobileconfig` (кандидат на слияние с `config-api`) |
| Рефералка / `bonus_bytes` | вне MVP |

Удаление кода — отдельным шагом реализации, не этим документом. Сохраняются:
**ADR-0005/0014** (EAP-MSCHAPv2/NT-hash — переутверждены спайком), **ADR-0020**
(Platega — но как тонкий webhook, пишущий `radcheck`, а не своя модель подписок).

---

## 1. What already exists (do not rebuild)

The MMVP is a working "smart VPN for RU": foreign traffic egresses abroad, RU traffic goes direct (ASN/GeoIP split). Verified live on iPhone 2026-06-18.

| Layer | Tech | Notes |
|---|---|---|
| IaC | **OpenTofu** + Ansible | Hetzner (egress + control-plane), Selectel/OpenStack (ingress). ADR-0009 (OSS-only). |
| Data plane | **sing-box** mesh (VLESS-Reality), **strongSwan** IKEv2 | client→ingress IPsec; ingress→egress Reality camouflage; ASN/GeoIP split in sing-box. ADR-0015, 0011, 0006. |
| Auth (VPN) | **FreeRADIUS** EAP-MSCHAPv2 | credential = `username` + **NT-hash** (MD4(UTF-16LE(pw)), not bcrypt — ADR-0014). Sticky `Framed-IP`. Revoke via `revoked_at`. ADR-0005. |
| Backend | **Go** services on **PostgreSQL 16** | ADR-0013. |
| → `config-api` | issues a VPN credential + Apple `.mobileconfig` on a paid Plati/Digiseller unique code | idempotent by `plati_order_id`. Live product **Digiseller 5937891**, `verify_url=/plati/issue`. ADR-0018. |
| → `orchestrator` | node registry (idempotent by `public_ip`) + health-checking (TLS probes, mark `down` after threshold) + admin API (`/healthz`, `/nodes`) | this is the **existing** orchestrator we extend. |
| → `db` | plain-SQL migrations (`0001_init`, `0002_nodes_public_ip_unique`) | 6 tables, see §4.1. |
| Client | Apple **IKEv2** via `.mobileconfig` | Android/Windows = phase 2. |
| Billing | **Plati/Digiseller** unique-code redemption *(MMVP only — MVP switches to platega.io, §7)* | ADR-0007, 0018. |
| License | **AGPL-3.0** | ADR-0010 — affects any third-party code you pull in. |

**Today there are no user accounts, no login, no cabinet.** Identity is `plati_buyer_id` (Digiseller buyer email); the flow is *buy code → redeem → download profile*. Subscriptions are **duration-based** (`30d`/`90d`/`365d`) with **no traffic cap** and **no enforced expiry revocation yet** (flagged as next step in MMVP-REPORT §4).

---

## 2. What this document adds (the delta)

The website introduces an **account layer** the MMVP doesn't have. Net-new work:

1. **Identity & sessions** — Telegram login + phone OTP (Twilio), JWT/refresh sessions, account deletion. (Today: none.)
2. **Authenticated, in-app credential issuance** — "Подключить устройство" issues an EAP credential for the logged-in user (today issuance is only triggered by a Digiseller code redirect).
3. **Traffic quota model** — packages are now **traffic-capped** (10 GB), not just time-based. `usage_log` already records consumption; **enforcement** (block when over quota) is new.
4. **Referrals** — invite link + bonus traffic.
5. **Notifications** — orchestrator pushes traffic/expiry events to the bot.
6. **Orchestrator cron duties** — expiry revocation + quota enforcement + node selection for configs.
7. **Payment provider** — MVP uses **platega.io** (payment intents + webhooks, binds to `user_id`). Digiseller's code-redemption stays only as the MMVP legacy path.
8. **New `account-api` service** — Go on the same Postgres; owns auth/account/billing endpoints (§9, §11).

Everything reuses the existing `users` / `subscriptions` / `auth_credentials` / `usage_log` / `nodes` tables and the `config-api` NT-hash + `.mobileconfig` machinery.

---

## 3. Decisions locked (from product)

| # | Topic | Decision |
|---|---|---|
| 1 | Stack | Go + PostgreSQL + existing services; OpenTofu/Ansible; sing-box/strongSwan/FreeRADIUS; Apple `.mobileconfig`. AGPL-3.0. |
| 2 | VPN tech | sing-box mesh + IKEv2 + RADIUS EAP-MSCHAPv2 (NT-hash). **Not** Hysteria2. |
| 3 | Trial | **300 MB AND 1 month** — whichever runs out first. |
| 4 | Paid package (MVP) | **300 ₽ → 10 GB (one-shot bucket), 1 month, unlimited devices.** MVP catalog is **month-only**. |
| 5 | Auto-renew | **No** auto-renew for now. |
| 6 | Payments | **platega.io** on MVP — СБП, RU bank card, international card; no crypto. (Digiseller is MMVP-only legacy.) |
| 7 | Multi-device | **Unlimited devices**, including on trial. |
| 8 | Referral | Qualifies on invitee's **first purchase**; reward **+1 GB to the inviter only** for now. Build the schema so **invitee discounts** can be added later (out of scope now). |
| 9 | Auth methods | **Phone and Telegram both.** SMS via **Twilio**. |
| 10 | Data residency / PII | **Not required.** Minimize stored data (project invariant: no user data on RU ingress). **Exception:** collect **email** for billing (platega.io receipts). Digiseller email is legacy. |
| 11 | Credentials | **Issue stable, long-lived creds.** Renewal extends the subscription; it does **not** rotate the password / reinstall the profile. |
| 12 | Quota period | **Monthly reset** — the 10 GB refills each month with the subscription. Exhausted before month end = blocked until the next period (or a top-up). |
| 13 | Identity merge | **Manual ("hand") merge** only; no automatic linking for now. |
| 14 | Bot transport | Orchestrator emits `notification_events`; the **bot is designed separately (out of scope now)** — push contract TBD. |
| 15 | Admin tooling | **Later.** |

---

## 4. Data model

PostgreSQL 16, plain-SQL migrations (matching `services/db/migrations/` convention). Add new tables/columns as migrations `0003+`; **do not break** `0001`/`0002`.

### 4.1 Existing tables (recap — `0001_init.up.sql`)

- **`users`**: `id`, `plati_buyer_id` UNIQUE, `email`, `created_at`, `status` (`active`/`blocked`/`churned`).
- **`subscriptions`**: `id`, `user_id`, `plati_order_id` UNIQUE (idempotency), `plan` (`30d`/`90d`/`365d`), `started_at`, `expires_at`, `amount_rub`, `status` (`active`/`expired`/`refunded`).
- **`auth_credentials`** *(= a "device")*: `id`, `user_id`, `username` UNIQUE, `nt_hash` (`^[0-9a-fA-F]{32}$`), `framed_ip` INET UNIQUE (pool `10.8.0.0/14`), `issued_at`, `revoked_at` (NULL = active), `last_used_at`.
- **`usage_log`**: `(user_id, date)` PK, `bytes_in`, `bytes_out` — daily traffic rollup.
- **`nodes`**: `id`, `role` (`ingress`/`egress`/`control`), `region`, `public_ip` UNIQUE, `status` (`up`/`down`/`draining`/`maintenance`), `last_heartbeat`, `deployed_at`, `config_version`.
- **`node_secrets`**: `(node_id, secret_type)` PK, encrypted `secret_value`.

### 4.2 Changes to existing tables

**`users`** — add identity + referral columns (keep `plati_buyer_id` for the legacy Digiseller path; web users may sign up before buying, so it's nullable):
| Column | Type | Notes |
|---|---|---|
| `email` | text NULL | already exists; now **collected for billing** (platega.io receipts), distinct from the legacy Digiseller email |
| `referral_code` | text UNIQUE | public, for the invite link |
| `referred_by` | uuid NULL FK→users.id | attribution, set at signup from `?ref=` |
| `bonus_bytes` | bigint NOT NULL DEFAULT 0 | accumulated referral bonus (see §6) |

**`subscriptions`** — add the monthly traffic quota (decisions #4, #12); MVP is month-only:
| Column | Type | Notes |
|---|---|---|
| `traffic_bytes_limit` | bigint NOT NULL | the **monthly quota**: `10 GB` paid, `300 MB` trial |
| `current_period_start` | timestamptz NOT NULL | start of the current monthly window; a cron advances it and the usage counter effectively resets |
| `kind` | text CHECK in (`trial`,`paid`) | trial vs paid pack |

> The `plan` CHECK currently allows `30d/90d/365d`. For MVP, settle on a single **month-only** plan (e.g. keep `30d`) and migrate the CHECK accordingly — one canonical value.
> Access is bounded by **`expires_at` (subscription end) AND the monthly quota** — hitting either blocks access until the next period / renewal (§6).

### 4.3 New tables

**`telegram_identities`** — `user_id` FK, `telegram_id` bigint UNIQUE, `username` text, `first_name`/`last_name`/`photo_url`, `linked_at`. *(Telegram is the cabinet's primary display identity.)*

**`phone_identities`** — `user_id` FK, `phone` text UNIQUE (E.164), `verified_at`.

**`otp_codes`** — `id`, `phone` (idx), `code_hash` (never store plaintext), `attempts` int, `expires_at` (~5 min), `consumed_at`, `request_ip` inet. For Twilio phone login.

**`sessions`** — `id`, `user_id`, `refresh_token_hash`, `user_agent`, `ip`, `expires_at`, `revoked_at`. Access tokens are stateless JWT; refresh tokens live here (rotating, revocable).

**`payments`** *(platega.io — MVP)* — `id`, `user_id` FK, `package_id`, `provider` (`platega`), `provider_payment_id` text UNIQUE, `amount_rub` int, `method` (`sbp`/`card`/`intl_card`), `status` (`created`/`pending`/`succeeded`/`failed`/`refunded`), `idempotency_key` text UNIQUE, `raw_payload` jsonb, `created_at`/`updated_at`. Entitlement changes happen **only** on a verified webhook (§7).

**`referrals`** — `id`, `inviter_user_id`, `invitee_user_id` UNIQUE, `status` (`pending`/`qualified`/`rewarded`), `qualifying_payment_id` uuid NULL FK→payments.id, `inviter_reward_bytes` bigint (1 GB), `invitee_reward_bytes` bigint DEFAULT 0 *(infra for future invitee discounts — decision #8)*, `qualified_at`, `rewarded_at`. Qualifies on the invitee's **first succeeded payment**.

**`notification_events`** *(consumed by the bot)* — `id`, `user_id`, `type` (`welcome`/`traffic_100mb`/`traffic_1gb`/`sub_expiring`/`sub_expired`), `payload` jsonb, `status` (`pending`/`sent`/`failed`), `dedupe_key` UNIQUE (fire-once per threshold/period), `created_at`, `sent_at`.

**`audit_log`** — `id`, `actor` (`user`/`system`), `user_id` NULL, `action`, `metadata` jsonb, `ip`, `created_at`. Record logins, device issue/revoke, payments, deletions.

> **Deliberately not stored** (decision #10, project invariant): no browsing history, no destination IPs/SNI, no connection logs, no payload. `usage_log` keeps only aggregate byte counters. Keep PII to phone + Telegram handle + Digiseller email.

---

## 5. Authentication & sessions (new)

### 5.1 Telegram login
Validate the Login-Widget `hash` (HMAC-SHA256, key = `SHA256(bot_token)`), reject stale `auth_date`. Upsert `users` + `telegram_identities` by `telegram_id`. Issue session.

### 5.2 Phone OTP (Twilio)
- `POST /auth/otp/request {phone}` → generate code, store **hash**, send via Twilio, rate-limit (§10). Prototype uses fixed `0000`.
- `POST /auth/otp/verify {phone, code}` → check hash/attempts/expiry, upsert `users` + `phone_identities`, issue session.

### 5.3 Tokens & account
- Access = short-lived JWT (`sub=user_id`); refresh = opaque, hashed in `sessions`, rotating.
- `POST /auth/refresh`, `POST /auth/logout`, `GET /me`, `DELETE /me`.
- **Account deletion** → revoke all `auth_credentials` (set `revoked_at`, RADIUS rejects next auth), revoke sessions, delete identities. Keep `subscriptions`/`amount_rub` for accounting per law; otherwise purge PII.

> A single `users` row may carry a Telegram identity, a phone identity, and a `plati_buyer_id` — link them, don't duplicate users. Define the merge rule when a phone user later buys via Digiseller with a new email (open item §13).

---

## 6. Entitlements: subscription & traffic quota

- `GET /subscription` → `{plan, active, expiresAt, autoRenew:false}`.
- `GET /traffic` → `{usedBytes, limitBytes}`.
- **`active`** = subscription `status='active'` AND `expires_at > now()`.
- **`limitBytes`** = `subscriptions.traffic_bytes_limit` of the active sub **+ `users.bonus_bytes`** (referral). `usedBytes` = sum of `usage_log.(bytes_in+bytes_out)` over the current period.
- **Trial** (decision #3): on first account creation, create a `trial` subscription, `traffic_bytes_limit = 300 MB`, `expires_at = now()+30d`, unlimited devices. Access ends when **either** the 300 MB is used **or** the month elapses.
- **Paid** (decision #4): `300 ₽ → 10 GB one-shot bucket, expires_at = +30 days, unlimited devices`. Buying again before expiry: extend `expires_at` and top up the bucket (define stacking rule — §13 resolved as one-shot, so simplest is "new pack replaces/extends").
- **Monthly reset** (decision #12): `usedBytes` is measured over the current period — `Σ usage_log` since `current_period_start`. A monthly cron advances `current_period_start`, which resets the effective counter (the 10 GB / 300 MB refills). When `usedBytes ≥ limitBytes` mid-period, block until the next period or a top-up. Access also ends at `expires_at`.
- **Referral bonus** (decision #8): on the invitee's first succeeded payment, `inviter.bonus_bytes += 1 GB`. This is a **one-time pool** that does **not** reset monthly (unlike the quota) — it's consumed after the monthly quota and carries across periods until used. No duration change. (`invitee_reward_bytes` reserved for later discounts.)
- **Renewal = stable creds** (decision #11): renewing extends `expires_at` / refills the bucket; the user's `auth_credentials` are **unchanged** (no password rotation, no profile reinstall).
- **Thresholds** → emit `notification_events` (dedup): `traffic_100mb` (≤100 MB left), `traffic_1gb` (≤1 GB left), `sub_expiring` (T-1 day), `sub_expired`. These power the design's "Рассылки об окончании трафика".

> **Quota enforcement is new.** Today nothing blocks an over-quota user. See §8 + §10 for how the orchestrator revokes/blocks credentials when `usedBytes ≥ limitBytes` or after expiry.

---

## 7. Billing — platega.io (MVP)

MVP switches off Digiseller's code-redemption to **platega.io** (decision #6), which binds payments to the logged-in `user_id` and supports СБП / RU card / international card. Needs a new ADR (payment provider for MVP).

1. `POST /packages/{id}/purchase` (authenticated, idempotency key) → create a `payments` row (`status=created`), create a payment at platega.io with `user_id` + `package_id` in the order metadata, return `{paymentUrl}`. Frontend redirects there.
2. **Webhook** `POST /webhooks/payments/platega`:
   - Verify the provider signature; look up by `provider_payment_id`; **idempotent** (safe to receive twice).
   - On `succeeded`: create/extend the `subscriptions` row for the bound `user_id` (`kind=paid`, `+30d`, `traffic_bytes_limit=10 GB`), set `current_period_start=now()` so the monthly quota starts fresh, and — if this is the user's first succeeded payment and they were `referred_by` someone — settle the referral (credit inviter `bonus_bytes += 1 GB`, mark `referrals.rewarded`).
   - On `failed`/`refunded`: update status; reverse grants on refund.
3. Entitlement changes happen **only** on a verified webhook — never on a client "success" redirect. Add a reconcile job for stuck `pending` payments.
4. No auto-renew (decision #5). Renewal does **not** rotate credentials (decision #11).

> **Legacy:** the existing `config-api` `GET /plati/issue?uniquecode=…` (Digiseller → EAP cred + `.mobileconfig`, idempotent by `plati_order_id`) stays for MMVP but is **not** used by the MVP website. Capture user email during the platega flow (decision #10) for receipts.

---

## 8. VPN credentials & the RADIUS data plane (integration)

A **"device" = an `auth_credentials` row**. "Подключить устройство" must, for the logged-in user:
1. Generate `username` + password → `nt_hash = NTHash(password)` (reuse `services/config-api/internal/credentials`).
2. Allocate a free `framed_ip` from `10.8.0.0/14`.
3. Insert `auth_credentials` (FreeRADIUS reads it for EAP-MSCHAPv2; no extra provisioning call — RADIUS queries the DB directly).
4. Render `.mobileconfig` (reuse `internal/mobileconfig`) embedding the EAP username/password and the chosen ingress endpoint.
5. Return the profile file (the frontend downloads it).

**Stable credentials** (decision #11): once issued, a device's `username`/`nt_hash` stay fixed for the life of the device — renewals never rotate them, so the installed `.mobileconfig` keeps working. (This drops the MMVP "renew = password rotation" behavior.)

**Unlimited devices** (decision #7): no cap on rows per user. **Revocation**: set `revoked_at` → the FreeRADIUS query template excludes it → next auth is rejected. Used for: account deletion, subscription expiry, over-quota. On top-up after a block, clear `revoked_at` (or issue is unnecessary — the same stable cred is simply re-enabled).

**Traffic accounting**: wire RADIUS Accounting (Acct-Input/Output-Octets) and/or sing-box stats into `usage_log` (backlog B1). The orchestrator reads `usage_log` to compute `usedBytes` and to trigger enforcement.

---

## 9. Orchestrator responsibilities

The existing orchestrator (node registry + health) gains the control-plane lifecycle the website needs. Keep the node-facing API; add cron/workers:

| Responsibility | Detail |
|---|---|
| Node registry + health *(exists)* | `POST/GET /nodes`, TLS probes, mark `down`. |
| **Node selection for configs** | Pick a healthy `ingress` node (by region/load) when issuing a `.mobileconfig`; provide failover targets so the client survives a blocked ingress ("Автосмена сервера"). Reserve capacity = extra healthy ingress nodes + Reality camouflage. No user-facing server picker. |
| **Expiry revocation** *(new — was a known gap)* | Cron: when `expires_at < now()`, mark `subscriptions.expired` and `revoked_at` on that user's credentials. |
| **Quota enforcement** *(new)* | When `usedBytes ≥ limitBytes`, revoke/suspend credentials; restore on top-up. |
| **Threshold detection** *(new)* | Scan usage vs limit/expiry → enqueue `notification_events`. |
| **Bot events** *(new — decision #14)* | Write `notification_events`; **dispatch is deferred** until the bot is designed separately. Keep the table + dedupe now; wire the push (likely `POST {BOT_URL}/notify`, HMAC) when the bot contract exists. |
| **Referral settlement** *(new)* | On the invitee's first succeeded payment, credit inviter `bonus_bytes`, write `referrals`. |

> **Service split (decision #4):** the repo's `orchestrator` stays the infra control plane (nodes, health, the cron duties above). Add a **new `account-api`** service (Go, same Postgres) for the account/auth/billing/device endpoints in §11. The legacy `config-api` is untouched (MMVP Digiseller path).

---

## 10. Background jobs & limits

| Job | Cadence | Action |
|---|---|---|
| Usage ingest | 1–5 min | RADIUS acct / sing-box stats → `usage_log`. |
| Quota + expiry sweep | 1–5 min / hourly | Compute period usage vs limit and `expires_at`; revoke creds when over/expired; re-enable when a new period frees quota; emit threshold events. |
| Monthly period roll | hourly check | Advance `current_period_start` when a month has elapsed (resets the effective quota counter). |
| Notification dispatch | *(deferred)* | Events accumulate in `notification_events`; dispatch wired when the bot is designed (decision #14). |
| Payment reconcile | 10 min | Poll platega.io for stuck `pending` payments (webhook fallback). |
| Node health | 30–60 s | Existing TLS probes. |
| OTP cleanup | hourly | Drop expired `otp_codes`. |

**Rate limits:** OTP request ≤ N/phone/hr and ≤ M/IP/hr; verify ≤ 5 attempts/code then invalidate. Throttle device issuance per user (even though devices are unlimited, prevent abuse).

---

## 11. Website-facing API (satisfies the frontend mock)

`/api/v1`, JSON, Bearer access token unless noted. Error envelope `{ "error": { "code", "message" } }`. Field names must match `src/lib/types.ts`.

| Method | Path | Auth | Returns / Notes |
|---|---|---|---|
| POST | `/auth/otp/request` | – | `{ok}` — Twilio SMS |
| POST | `/auth/otp/verify` | – | `{accessToken, refreshToken, user}` |
| POST | `/auth/telegram` | – | `{accessToken, refreshToken, user}` |
| POST | `/auth/refresh` | refresh | new tokens |
| POST | `/auth/logout` | yes | `204` |
| GET | `/me` | yes | `{id, telegramUsername, phone, referralCode}` |
| DELETE | `/me` | yes | `204` — revokes creds + sessions |
| GET | `/subscription` | yes | `{plan, active, expiresAt, autoRenew:false}` |
| GET | `/traffic` | yes | `{usedBytes, limitBytes}` |
| GET | `/packages` | yes | catalog (MVP: one 300 ₽ / 10 GB pack) |
| POST | `/packages/{id}/purchase` | yes | `{paymentUrl}` → platega.io |
| GET | `/referral` | yes | `{link, invitedCount, rewardedCount}` |
| POST | `/devices` | yes | issues credential → `{config(.mobileconfig), subscriptionInfo}` |
| GET | `/devices` | yes | list (active creds) |
| DELETE | `/devices/{id}` | yes | revoke (`revoked_at`) |

**Provider/internal-facing:** `POST /webhooks/payments/platega` (signed — MVP billing source of truth); `GET /plati/issue?uniquecode=…` (Digiseller — MMVP legacy, not used by MVP); `GET /healthz`; orchestrator `GET/POST /nodes`.
**Deferred:** `POST {BOT_URL}/notify` (orchestrator → bot, HMAC) — pending bot design (decision #14).

---

## 12. Non-functional requirements

- **Data minimization** (decision #10 + invariant): store only phone, Telegram handle, Digiseller email, aggregate bytes. No connection/destination logs. No 152-ФЗ residency program needed, but keep secrets out of the repo (already enforced via `.gitignore`/ansible-vault).
- **Security:** NT-hash for EAP (never store plaintext VPN passwords beyond the issuance response); hash OTPs; secrets in ansible-vault now / orchestrator-distributed in phase 2 (ADR-0012); **verify platega.io webhook signatures**; HMAC on bot calls (deferred); per-device revocable creds distinct from web sessions.
- **Idempotency:** platega payments keyed by `idempotency_key` + `provider_payment_id` (webhook safe to repeat); legacy Digiseller issuance keyed by `plati_order_id`; bot events deduped by `dedupe_key`.
- **Observability** (backlog B1): Prometheus + node/strongSwan/sing-box exporters; metrics for active creds, traffic ingested, payment redemptions, bot push failures, node health.
- **Licensing:** AGPL-3.0 — any backend dependency/SaaS-exposed change must be compatible (ADR-0010).

---

## 13. Open items / to confirm

The original §13 questions are now resolved in §3 (trial = 300 MB + 1 month; month-only plan; platega.io; `account-api`; manual identity merge; referral on first purchase, inviter-only with invitee infra; stable creds; one-shot bucket). Remaining:

1. **platega.io integration** — drafted in [`docs/adr/0020-platega-io-payments-mvp.md`](adr/0020-platega-io-payments-mvp.md). Remaining to confirm against the live merchant account: exact create-transaction path, whether the webhook carries a separate body HMAC beyond `X-Secret`, the precise `status` set, and RU-card method id (10 vs 11).
2. **Bot push contract** — URL, payload schema, HMAC secret for delivering `notification_events`. Deferred until the bot is designed (decision #14); `notification_events` is built now so nothing is lost.
3. **Email capture point** — exactly where in the platega flow we collect/confirm the billing email (decision #10), and whether it's required before first purchase.
4. **Top-up / re-buy mid-period** — buying again before expiry: extend `expires_at` and restart the monthly window, or stack a second quota onto the current period? (Confirm desired behavior.)
5. **Bonus-pool accounting** — exact rule for how `bonus_bytes` is drawn down relative to the monthly quota and carried across period resets.

---

## 14. Assumptions

- Go services on PostgreSQL 16; new work as migrations `0003+` in a new **`account-api`** service (auth/account/billing/devices); `orchestrator` gains cron duties; `config-api` left as-is (MMVP).
- VPN remains sing-box/strongSwan/FreeRADIUS EAP-MSCHAPv2; a "device" is an `auth_credentials` row + `.mobileconfig` with **stable** creds; clients are Apple IKEv2 for MVP (Android/Windows phase 2).
- Payments on MVP via **platega.io** (webhook-driven, bound to `user_id`); methods СБП / RU card / international card; **no auto-renew**, **no crypto**. Digiseller remains MMVP-only legacy.
- Telegram + phone (Twilio) auth; unlimited devices incl. trial; **trial = 300 MB + 1 month**; paid **300 ₽ / 10 GB per month (resets monthly) / 1 month**; referral **+1 GB to inviter on invitee's first purchase**.
- Minimal PII (no connection logs); **email collected for billing**; identity merges are manual; bot delivery deferred; admin tooling deferred.
- Domain `flynet.pro`; invite link `https://flynet.pro/?ref=<referral_code>`.
- API field names match the existing frontend (`src/lib/types.ts`) so no client changes are needed.
