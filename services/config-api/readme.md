# config-api

Go-сервис (ADR-0013). Точка интеграции с Plati и выдачи клиентского конфига.

Ответственность (Plati = Digiseller, ADR-0018):
1. `GET /plati/issue?uniquecode=...` — валидация уникального кода через Digiseller API
   (токен `POST /api/apilogin`, `sign=SHA256(api_key+timestamp)`; затем
   `GET /api/purchases/unique-code/{code}?token=...`).
2. Маппинг `id_goods → план`; создание юзера/подписки в PostgreSQL (идемпотентно по
   `uniquecode`; покупатель — по email). Схема — `../db`.
3. Генерация EAP-кредов и запись в `auth_credentials`: username + NT-hash пароля
   (MD4(UTF-16LE), hex) — для MSCHAPv2, не bcrypt (ADR-0014).
4. Генерация `.mobileconfig` из шаблона и возврат покупателю.

Креды Digiseller опциональны: без них сервис стартует, `/plati/issue` отдаёт 503.
На боевом аккаунте Plati это товар `uniqueunfixed` + `content_type=DigisellerCode` с
`verify_code.auto_verify` и `verify_url=https://api.fly-vpn.net/plati/issue` (товар 5937891,
ADR-0018). Чек и уникальный код покупателю шлёт сам Digiseller на почту; наш сервер отдаёт
`.mobileconfig` по `verify_url` — собственный мейлер не нужен. Креды на деплое приходят из
GitHub-секрета `CONTROL_DIGISELLER` (не из `CONTROL_VAULT`).

## Шаблон профиля
`profile.mobileconfig.tmpl` — Apple Configuration Profile для IKEv2 + EAP-MSCHAPv2,
языко-независимый (токены `{{NAME}}` подставляет сервис). Проверен на well-formed XML.

Токены:

| Токен                     | Значение |
|---------------------------|----------|
| `{{PROFILE_IDENTIFIER}}`  | reverse-DNS id профиля (напр. `com.X.vpn.<uuid>`) |
| `{{PROFILE_UUID}}`        | UUID профиля |
| `{{PAYLOAD_UUID}}`        | UUID VPN-payload |
| `{{DISPLAY_NAME}}`        | отображаемое имя (напр. «Smart Internet») |
| `{{ORG_NAME}}`            | организация |
| `{{VPN_REMOTE_ADDRESS}}`  | адрес ingress (`vpn.X.com`) |
| `{{VPN_REMOTE_IDENTIFIER}}` | идентификатор сервера (CN/домен серт.) |
| `{{SERVER_CA_CN}}`        | CN издателя серверного сертификата |
| `{{EAP_USERNAME}}`        | EAP username |
| `{{EAP_PASSWORD}}`        | EAP password |

OnDemand (авто-подключение) — phase 2, в шаблоне не задаётся.

## Структура (Go)
- `cmd/config-api` — точка входа (config → pgx → HTTP).
- `internal/config` — конфиг из окружения.
- `internal/credentials` — генерация username/password и **NT-hash** (`MD4(UTF-16LE)`).
- `internal/digiseller` — клиент Digiseller API (токен с кэшем + проверка уникального кода).
- `internal/mobileconfig` — рендер профиля из шаблона (XML-экранирование, проверка well-formed).
- `internal/store` — pgx: идемпотентная по `plati_order_id` (=uniquecode) выдача.
- `internal/httpapi` — `GET /healthz`, `GET /plati/issue`.
- `template.go` — `go:embed` шаблона профиля.

## Конфигурация (env)
Обяз.: `CONFIGAPI_DATABASE_URL`, `CONFIGAPI_VPN_REMOTE`.
ACME: `CONFIGAPI_ACME_DOMAIN`, `CONFIGAPI_ACME_CACHE`.
Digiseller (опц.): `CONFIGAPI_DIGISELLER_SELLER_ID`, `CONFIGAPI_DIGISELLER_API_KEY`,
`CONFIGAPI_PLAN_BY_GOODS` (`id:30d,id:365d`), `CONFIGAPI_DEFAULT_PLAN` (`30d`).
Прочее: `CONFIGAPI_VPN_REMOTE_ID`, `CONFIGAPI_SERVER_CA_CN`, `CONFIGAPI_LISTEN`,
`CONFIGAPI_ORG`, `CONFIGAPI_DISPLAY_NAME`.

## Проверено
`go build/vet/test ./...` зелёные. NT-hash сверен с известными векторами (тот же MD4,
что ждёт FreeRADIUS). Store — интеграционный тест на PostgreSQL 16 (идемпотентность,
аллокация sticky-IP, ротация nt_hash). End-to-end: выданный кред аутентифицируется через
FreeRADIUS (Access-Accept + Framed-IP). Интеграционный тест store включается переменной
`CONFIGAPI_TEST_DSN` (без неё — пропуск).

## Известные TODO (честно)
- **Семантика `unique_code_state.state`** (Digiseller, 1–5) — уточнить множество «валидных»
  статусов по боевому аккаунту; сейчас код принимается, если `id_goods > 0` (ADR-0018).
- **Продление = ротация пароля**: на повторную покупку username/IP стабильны, но nt_hash
  ротируется под свежий пароль → юзер переустанавливает профиль (MMVP-поведение; альтернатива —
  не переиздавать профиль при продлении — на потом).
- HTTPS/TLS терминируется перед сервисом (reverse-proxy/LB), не в самом config-api на MMVP.
