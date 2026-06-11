<dir name="control-plane" role="ansible-role-control-plane">
  <readme href="./readme.md"/>
  <purpose>Стек control plane: PostgreSQL + FreeRADIUS + Go-сервисы config-api (TLS/ACME) и orchestrator.</purpose>
  <invariants>
    <i>FreeRADIUS ходит в БД read-only ролью с SELECT только на auth_credentials.</i>
    <i>Auth: NT-Password из auth_credentials, отзыв по revoked_at; sticky Framed-IP.</i>
    <i>Секреты (пароли БД, plati_secret, clients) только из vault; в репозитории их нет — роль падает без них.</i>
    <i>Шаблон sql.j2 содержит FreeRADIUS-xlat %{...}; статичная часть обёрнута в {% raw %} для Jinja.</i>
    <i>config-api/orchestrator собираются на узле (пин Go); env из vault; config-api сам терминирует TLS (ACME).</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">PostgreSQL/FreeRADIUS, миграции, sql-модуль, сборка+деплой Go-сервисов</e>
    <e path="./templates/sql.j2">SQL-модуль FreeRADIUS (проверен на 3.2.5)</e>
    <e path="./templates/config-api.service.j2">systemd config-api (:80/:443 ACME)</e>
    <e path="./templates/orchestrator.service.j2">systemd orchestrator</e>
    <e path="./readme.md">состав, проверка, зависимости, порты</e>
  </entrypoints>
  <depends-on>
    <d>../../../../services/db</d>
    <d>../../../../services/config-api</d>
    <d>../../../../services/orchestrator</d>
    <d>../../../terraform/modules/control-plane</d>
  </depends-on>
</dir>
