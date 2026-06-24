<dir name="account-api" role="backend-service">
  <readme href="./readme.md"/>
  <purpose>Backend веб-кабинета MVP: аккаунты/сессии, подписки и месячная traffic-quota, выдача устройств (.mobileconfig), платежи Platega.io.</purpose>
  <invariants>
    <i>Права/подписки меняются ТОЛЬКО по подтверждённому вебхуку Platega (ADR-0020); идемпотентно по provider_payment_id + idempotency_key.</i>
    <i>Креды стабильны (decision #11): продление не ротирует username/nt_hash; устройств без лимита.</i>
    <i>NT-hash = MD4(UTF-16LE(pw)) — совпадает с config-api байт-в-байт (FreeRADIUS/ADR-0014). credentials/mobileconfig продублированы намеренно (cross-module internal import невозможен).</i>
    <i>config-api не трогаем — это отдельный сервис для веб-продукта (§9).</i>
    <i>Секреты (JWT, Platega, Twilio, telegram bot token) — из окружения/vault, не из репозитория.</i>
    <i>PII по минимуму: телефон + Telegram-handle + billing-email; никаких логов соединений.</i>
  </invariants>
  <entrypoints>
    <e path="./cmd/account-api/main.go">точка входа сервиса</e>
    <e path="./internal/httpapi">маршруты /api/v1 + вебхук Platega + auth-middleware</e>
    <e path="./internal/store">доступ к PostgreSQL (identity, session, otp, entitlement, device, billing)</e>
    <e path="./internal/platega">клиент Platega.io (ADR-0020)</e>
    <e path="./internal/auth">валидация Telegram login-widget + OTP/Twilio</e>
    <e path="./internal/token">access-JWT + opaque refresh</e>
  </entrypoints>
  <depends-on>
    <d>../db</d>
    <d>../../docs/backend-requirements.md</d>
    <d>../../docs/adr/0020-platega-io-payments-mvp.md</d>
    <d>../../docs/adr/0014-eap-mschapv2-password-storage.md</d>
  </depends-on>
</dir>
