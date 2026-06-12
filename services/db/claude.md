<dir name="db" role="data-layer">
  <readme href="./readme.md"/>
  <purpose>Схема PostgreSQL control plane как plain-SQL миграции (up/down).</purpose>
  <invariants>
    <i>Языко- и раннер-независимо: обычный SQL, не привязан к выбору языка сервисов (ADR-0013).</i>
    <i>nt_hash — NT-hash (MD4) для MSCHAPv2, не bcrypt (ADR-0014); node_secrets.secret_value — зашифровано вне БД.</i>
    <i>Идентификация юзера — по plati_buyer_id, не по username/email.</i>
    <i>plati_order_id (=uniquecode Digiseller) уникален — идемпотентность повторной выдачи.</i>
  </invariants>
  <entrypoints>
    <e path="./migrations/0001_init.up.sql">исходная схema (README §4)</e>
    <e path="./migrations/0001_init.down.sql">откат</e>
  </entrypoints>
  <depends-on>
    <d>../../docs/adr/0013-backend-language-open.md</d>
  </depends-on>
</dir>
