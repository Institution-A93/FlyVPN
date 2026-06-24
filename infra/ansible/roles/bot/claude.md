<dir name="bot" role="ansible-role-bot">
  <readme href="./readme.md"/>
  <purpose>Деплой Telegram-бота (services/bot, Python/aiogram — ADR-0019) на control plane: venv + systemd, long-polling.</purpose>
  <invariants>
    <i>bot_token и секреты только из vault (ADR-0012); роль падает без bot_token.</i>
    <i>Long-polling: входящих публичных портов нет; /notify только на localhost.</i>
    <i>Бот автономен — не лезет в БД/RADIUS/config-api (выдача профиля — mock до интеграции).</i>
    <i>Python-рантайм ставится только для бота (исключение из Go, ADR-0019); остальное на узле — Go.</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">python venv + код + EnvironmentFile + systemd</e>
    <e path="./templates/flyvpn-bot.service.j2">systemd-юнит (long-polling)</e>
    <e path="./templates/bot.env.j2">env из vault</e>
    <e path="./readme.md">состав, хостинг, секреты, порты</e>
  </entrypoints>
  <depends-on>
    <d>../../../../services/bot</d>
  </depends-on>
</dir>
