<dir name="bot" role="service-telegram-bot">
  <readme href="./readme.md"/>
  <purpose>Telegram-бот: онбординг, выдача профиля, уведомления. Python/aiogram (ADR-0019, исключение из Go-инварианта ADR-0013).</purpose>
  <invariants>
    <i>Автономен на старте: in-memory mock, не трогает control plane (БД/config-api).</i>
    <i>BOT_TOKEN и секреты — только из окружения (.env локально / EnvironmentFile из vault), не в репозитории.</i>
    <i>Long-polling: входящих публичных портов нет; /notify слушает только localhost.</i>
    <i>Выдача .mobileconfig сейчас mock; реальные EAP-креды появляются только через config-api (TODO).</i>
  </invariants>
  <entrypoints>
    <e path="./bot/main.py">точка входа (polling + /notify)</e>
    <e path="./bot/handlers.py">команды и callbacks</e>
    <e path="./bot/vpnconfig.py">mock .mobileconfig (→ config-api позже)</e>
    <e path="./bot/notifications.py">отправка + POST /notify (пуш от orchestrator)</e>
    <e path="./readme.md">запуск, деплой, структура, TODO</e>
  </entrypoints>
  <depends-on>
    <d>../../infra/ansible/roles/bot</d>
  </depends-on>
</dir>
