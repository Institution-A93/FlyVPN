# services/account-api

Backend веб-кабинета FLY VPN (MVP): аккаунты, сессии, подписки/квота трафика, выдача
устройств и платежи Platega.io. Go + PostgreSQL (общая БД с `config-api`/`orchestrator`).
Полная спека — [`docs/backend-requirements.md`](../../docs/backend-requirements.md);
платежи — [ADR-0020](../../docs/adr/0020-platega-io-payments-mvp.md).

> `config-api` (legacy Digiseller unique-code, ADR-0018) остаётся нетронутым. `account-api` —
> отдельный сервис для веб-продукта (ADR-0013: Go). NT-hash и шаблон `.mobileconfig`
> намеренно продублированы из `config-api` (cross-module import `internal/` невозможен,
> а NT-hash должен совпадать байт-в-байт для FreeRADIUS).

## Эндпоинты (`/api/v1`, Bearer access-JWT, ошибки `{"error":{"code","message"}}`)
| Метод | Путь | Назначение |
|---|---|---|
| POST | `/auth/otp/request` | OTP по SMS (Twilio; prototype — код `0000`) |
| POST | `/auth/otp/verify` | вход по телефону → `{accessToken, refreshToken, user}` |
| POST | `/auth/telegram` | вход по Telegram login-widget (HMAC-проверка) |
| POST | `/auth/refresh` | ротация refresh-сессии |
| POST | `/auth/logout` | отзыв сессии |
| GET/DELETE | `/me` | профиль / удаление аккаунта (отзыв кредов+сессий, чистка PII) |
| GET | `/subscription` | `{plan, active, expiresAt, autoRenew:false}` |
| GET | `/traffic` | `{usedBytes, limitBytes}` (квота периода + реф-бонус) |
| GET | `/packages` | каталог (MVP: один пакет 300 ₽ / 10 ГБ / 1 мес) |
| POST | `/packages/{id}/purchase` | создаёт платёж Platega → `{paymentUrl}` |
| GET | `/referral` | `{link, invitedCount, rewardedCount}` |
| POST/GET | `/devices` | выдать кред + `.mobileconfig` / список устройств |
| DELETE | `/devices/{id}` | отзыв устройства (`revoked_at`) |
| POST | `/webhooks/payments/platega` | callback Platega — **единственный** источник прав |
| GET | `/healthz` | проверка живости (+ ping БД) |

## Модель прав (docs/backend-requirements.md §6)
- Триал при первом входе: **300 МБ + 1 мес**.
- Платный пакет: **300 ₽ → 10 ГБ/мес**, продление НЕ ротирует креды (стабильные креды).
- Доступ ограничен `expires_at` **И** месячной квотой; счётчик периода — `Σ usage_log`
  с `current_period_start`. Реф-бонус (`+1 ГБ` инвайтеру) — отдельный пул сверх квоты.
- Права меняются **только** по подтверждённому вебхуку Platega (ADR-0020), идемпотентно.

## Конфигурация (env; секреты — из vault/оркестратора, не из репозитория)
Обязательные: `ACCOUNTAPI_DATABASE_URL`, `ACCOUNTAPI_VPN_REMOTE`, `ACCOUNTAPI_JWT_SECRET`.
Опциональные: `ACCOUNTAPI_LISTEN` (`:8080`), `ACCOUNTAPI_PUBLIC_BASE_URL` (`https://flynet.pro`),
`ACCOUNTAPI_TELEGRAM_BOT_TOKEN`, `ACCOUNTAPI_TWILIO_*` (без них OTP = `0000`),
`ACCOUNTAPI_PLATEGA_*` (без них purchase/webhook → 503), `ACCOUNTAPI_ACCESS_TTL`,
`ACCOUNTAPI_REFRESH_TTL`.

## Сборка и тесты
```sh
go build ./... && go vet ./...
go test ./...                                  # без БД — unit-тесты (telegram HMAC, NT-hash, JWT)
ACCOUNTAPI_TEST_DSN=postgres://… go test ./... # интеграционные (схема 0001..0004)
```
Проверено на PostgreSQL 16: сценарии signup→trial→device, purchase→webhook→paid+referral,
ротация сессий, и e2e по HTTP (OTP-login → /me → /subscription → /traffic → /devices).
