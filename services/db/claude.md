<dir name="db" role="data-layer">
  <readme href="./readme.md"/>
  <purpose>Схема PostgreSQL control plane как plain-SQL миграции (up/down).</purpose>
  <invariants>
    <i>Языко- и раннер-независимо: обычный SQL, не привязан к выбору языка сервисов (ADR-0013).</i>
    <i>nt_hash — NT-hash (MD4) для MSCHAPv2, не bcrypt (ADR-0014); node_secrets.secret_value — зашифровано вне БД.</i>
    <i>Идентификация юзера — по plati_buyer_id, не по username/email.</i>
    <i>plati_order_id (=uniquecode Digiseller) уникален — идемпотентность повторной выдачи.</i>
    <i>Новые миграции (0003+) не ломают 0001/0002: NOT NULL-колонки добавляются с бэкфиллом legacy-строк.</i>
    <i>Права/подписки MVP меняются только по подтверждённому вебхуку Platega (ADR-0020) — на уровне сервиса, не БД.</i>
  </invariants>
  <entrypoints>
    <e path="./migrations/0001_init.up.sql">исходная схема MMVP (README §4)</e>
    <e path="./migrations/0003_accounts_quota.up.sql">MVP: ALTER users/subscriptions (рефералы + месячная traffic-quota)</e>
    <e path="./migrations/0004_identities_billing.up.sql">MVP: новые таблицы (identities, sessions, payments, referrals, notifications, audit)</e>
  </entrypoints>
  <depends-on>
    <d>../../docs/adr/0013-backend-language-open.md</d>
    <d>../../docs/adr/0020-platega-io-payments-mvp.md</d>
    <d>../../docs/backend-requirements.md</d>
  </depends-on>
</dir>
