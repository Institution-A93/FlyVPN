<dir name="config-api" role="service-config-api">
  <readme href="./readme.md"/>
  <purpose>Plati-вебхук, генерация .mobileconfig, выдача EAP-кредов.</purpose>
  <invariants>
    <i>HMAC-подпись Plati валидируется до любых изменений в БД.</i>
    <i>username — случайная строка, не email; формат хранения пароля — ADR-0014 (не bcrypt при EAP-MSCHAPv2).</i>
    <i>plati_order_id уникален — идемпотентность повторного вебхука.</i>
  </invariants>
  <entrypoints>
    <e path="./profile.mobileconfig.tmpl">шаблон Apple-профиля (IKEv2/EAP-MSCHAPv2)</e>
    <e path="./readme.md">ответственность, токены шаблона, flow</e>
  </entrypoints>
  <depends-on>
    <d>../db</d>
    <d>../orchestrator</d>
  </depends-on>
</dir>
