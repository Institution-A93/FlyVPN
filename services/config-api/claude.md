<dir name="config-api" role="service-config-api">
  <readme href="./readme.md"/>
  <purpose>Go-сервис: Plati-вебхук, генерация .mobileconfig, выдача EAP-кредов (ADR-0013).</purpose>
  <invariants>
    <i>HMAC-подпись Plati валидируется до любых изменений в БД (internal/plati, constant-time).</i>
    <i>username — случайная строка, не email; пароль хранится как NT-hash (MD4), не bcrypt (ADR-0014).</i>
    <i>plati_order_id уникален — идемпотентность повторного вебхука (internal/store).</i>
    <i>Секретов в коде нет; конфиг из окружения (CONFIGAPI_*).</i>
  </invariants>
  <entrypoints>
    <e path="./cmd/config-api">точка входа (main)</e>
    <e path="./internal/credentials">username/password + NT-hash</e>
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
