<dir name="services" role="backend-services">
  <readme href="./readme.md"/>
  <purpose>Backend control plane: единый Go-бинарь control (ADR-0021) + bot (Python/aiogram, ADR-0019) + db (миграции). Доступ — нативный FreeRADIUS на PostgreSQL, не сервис.</purpose>
  <invariants>
    <i>Тонкий слой логики поверх готовых компонентов; самописное минимизируем.</i>
    <i>Общая БД — PostgreSQL на control plane; схема в ./db (миграции).</i>
    <i>Язык — Go (ADR-0013); единственное исключение — ./bot (Python/aiogram, ADR-0019).</i>
    <i>Единственный писатель в auth_credentials — пакет contract в control; панель и коннекторы ходят через него.</i>
  </invariants>
  <entrypoints>
    <e path="./db">схема PostgreSQL (миграции; auth_credentials + radacct — движок entitlement)</e>
    <e path="./control">единый бинарь: contract/profiles/fleet/panel + webhook-коннекторы (ADR-0021)</e>
    <e path="./bot">Telegram-бот: онбординг, доставка профиля; зовёт API control</e>
  </entrypoints>
  <depends-on>
    <d>../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
