<dir name="services" role="backend-services">
  <readme href="./readme.md"/>
  <purpose>Backend-сервисы control plane: config-api, orchestrator (Go, ADR-0013) и bot (Python/aiogram, ADR-0019).</purpose>
  <invariants>
    <i>Тонкий слой логики поверх готовых компонентов; самописное минимизируем.</i>
    <i>Общая БД — PostgreSQL на control plane; схема в ./db (миграции).</i>
    <i>Язык — Go (ADR-0013); единственное исключение — ./bot (Python/aiogram, ADR-0019).</i>
  </invariants>
  <entrypoints>
    <e path="./db">схема PostgreSQL (миграции)</e>
    <e path="./config-api">выдача по коду Plati/Digiseller и генерация .mobileconfig</e>
    <e path="./orchestrator">управление узлами</e>
    <e path="./bot">Telegram-бот: онбординг, выдача профиля, уведомления</e>
  </entrypoints>
  <depends-on>
    <d>../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
