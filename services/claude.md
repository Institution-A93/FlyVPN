<dir name="services" role="backend-services">
  <readme href="./readme.md"/>
  <purpose>Backend-сервисы control plane: config-api (MMVP legacy), account-api (веб-кабинет MVP), orchestrator (Go, ADR-0013) и bot (Python/aiogram, ADR-0019).</purpose>
  <invariants>
    <i>Тонкий слой логики поверх готовых компонентов; самописное минимизируем.</i>
    <i>Общая БД — PostgreSQL на control plane; схема в ./db (миграции).</i>
    <i>Язык — Go (ADR-0013); единственное исключение — ./bot (Python/aiogram, ADR-0019).</i>
    <i>config-api (Digiseller, ADR-0018) — MMVP-legacy, не трогаем; веб-продукт MVP обслуживает account-api (платежи Platega, ADR-0020).</i>
  </invariants>
  <entrypoints>
    <e path="./db">схема PostgreSQL (миграции)</e>
    <e path="./config-api">выдача по коду Plati/Digiseller и генерация .mobileconfig (MMVP legacy)</e>
    <e path="./account-api">веб-кабинет MVP: аккаунты, подписки/квота, устройства, платежи Platega</e>
    <e path="./orchestrator">управление узлами + cron-обязанности MVP</e>
    <e path="./bot">Telegram-бот: онбординг, выдача профиля, уведомления</e>
  </entrypoints>
  <depends-on>
    <d>../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
