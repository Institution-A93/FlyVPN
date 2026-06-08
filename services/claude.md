<dir name="services" role="backend-services">
  <readme href="./readme.md"/>
  <purpose>Go-сервисы control plane: config-api и orchestrator.</purpose>
  <invariants>
    <i>Тонкий слой логики поверх готовых компонентов; самописное минимизируем.</i>
    <i>Общая БД — PostgreSQL на control plane (схема — в README §4).</i>
  </invariants>
  <entrypoints>
    <e path="./config-api">Plati-вебхук и генерация .mobileconfig</e>
    <e path="./orchestrator">управление узлами</e>
  </entrypoints>
  <depends-on>
    <d>../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
