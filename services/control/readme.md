# control

Единый Go-бинарь control plane (ADR-0021, модульный монолит). Поглотил прежние
config-api и orchestrator; account-api растворён. Один pgx-пул, один HTTP-сервер
(внутренний), одна фоновая горутина health.

## Пакеты

| Пакет | За что |
|-------|--------|
| `contract` | **Единственный писатель** `auth_credentials`: Provision/Renew/Revoke/Get. nt-hash стабильный; продление двигает срок + обнуляет period (без переноса). Usage/State (active/lapsed/revoked). |
| `profiles` | Рендер `.mobileconfig` (iOS) и `.sswan` (Android strongSwan-app). |
| `fleet` | Реестр узлов, health-пробы, CoA-origination (Disconnect на DAE-порт узла). |
| `panel` | Тонкая HTML-панель для нетех-суппорта (cookie-вход по operator-токену). |
| `httpapi` | Операторский REST (bearer) + платёжный webhook (по секрету) → contract. |
| `credentials`/`store`/`config` | NT-hash + ген; pgx-пул; конфиг из env. |

## Запуск

```
CONTROL_DSN=postgres://… CONTROL_OPERATOR_TOKEN=… CONTROL_SERVER_ADDR=vpn.example.net \
  go run ./cmd/control          # слушает CONTROL_LISTEN (по умолч. 127.0.0.1:8080)
```
Панель: `/panel` (вход по operator-токену). API: `/api/v1/users/{id}/{provision,renew,revoke}`,
`/webhooks/pay`, `/fleet/nodes`, `/healthz`.

## Тесты

`go test ./...` — юнит (profiles); интеграционные (contract, fleet) идут при заданном
`CONTROL_TEST_DSN` (PostgreSQL со схемой `services/db` 0001+0006), иначе скипаются.

## Анти-локаут (ADR-0021)

Доступ к Telegram/оплате остаётся всегда: срок/кап не режут auth, lapsed-юзер попадает
в restricted-пул (walled garden) — логика в FreeRADIUS `authorize_reply` (см.
`infra/.../control-plane/templates/sql.j2`), не здесь. Контракт пишет только данные.
