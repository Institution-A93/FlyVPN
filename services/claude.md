<dir name="services" role="backend-services">
  <readme href="./readme.md"/>
  <purpose>Backend-сервисы control plane: config-api и orchestrator (язык — ADR-0013).</purpose>
  <invariants>
    <i>Тонкий слой логики поверх готовых компонентов; самописное минимизируем.</i>
    <i>Общая БД — PostgreSQL на control plane; схема в ./db (миграции).</i>
  </invariants>
  <entrypoints>
    <e path="./db">схема PostgreSQL (миграции)</e>
    <e path="./config-api">выдача по коду Plati/Digiseller и генерация .mobileconfig</e>
    <e path="./orchestrator">управление узлами</e>
  </entrypoints>
  <depends-on>
    <d>../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
