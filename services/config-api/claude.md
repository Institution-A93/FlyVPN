<dir name="config-api" role="go-service-config-api">
  <readme href="./readme.md"/>
  <purpose>Plati-вебхук, генерация .mobileconfig, выдача EAP-кредов.</purpose>
  <invariants>
    <i>HMAC-подпись Plati валидируется до любых изменений в БД.</i>
    <i>Пароль хранится только как bcrypt-хеш; username — случайная строка, не email.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">ответственность и flow</e>
  </entrypoints>
  <depends-on>
    <d>../orchestrator</d>
  </depends-on>
</dir>
