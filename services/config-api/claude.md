<dir name="config-api" role="service-config-api">
  <readme href="./readme.md"/>
  <purpose>Go-сервис: выдача по уникальному коду Plati/Digiseller, генерация .mobileconfig, EAP-креды (ADR-0013/0018).</purpose>
  <invariants>
    <i>Уникальный код валидируется через Digiseller API до любых изменений в БД (internal/digiseller).</i>
    <i>username — случайная строка, не email; пароль хранится как NT-hash (MD4), не bcrypt (ADR-0014).</i>
    <i>plati_order_id (=uniquecode) уникален — идемпотентность повторного запроса (internal/store).</i>
    <i>Креды Digiseller опциональны; без них /plati/issue -> 503. Секретов в коде нет (CONFIGAPI_*).</i>
  </invariants>
  <entrypoints>
    <e path="./cmd/config-api">точка входа (main)</e>
    <e path="./internal/credentials">username/password + NT-hash</e>
    <e path="./internal/digiseller">клиент Digiseller (токен + проверка кода)</e>
    <e path="./internal/store">pgx: идемпотентная выдача</e>
    <e path="./internal/httpapi">/healthz, /plati/issue</e>
    <e path="./profile.mobileconfig.tmpl">шаблон Apple-профиля (IKEv2/EAP-MSCHAPv2)</e>
    <e path="./readme.md">структура, конфиг, проверка, TODO</e>
  </entrypoints>
  <depends-on>
    <d>../db</d>
    <d>../orchestrator</d>
  </depends-on>
</dir>
