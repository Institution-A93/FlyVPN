<dir name="control-plane" role="ansible-role-control-plane">
  <readme href="./readme.md"/>
  <purpose>Стек control plane: PostgreSQL + FreeRADIUS (EAP-MSCHAPv2/NT-hash). config-api/orchestrator — позже (ADR-0013).</purpose>
  <invariants>
    <i>FreeRADIUS ходит в БД read-only ролью с SELECT только на auth_credentials.</i>
    <i>Auth: NT-Password из auth_credentials, отзыв по revoked_at; sticky Framed-IP.</i>
    <i>Секреты (пароли БД, clients) только из vault; в репозитории их нет — роль падает без них.</i>
    <i>Шаблон sql.j2 содержит FreeRADIUS-xlat %{...}; статичная часть обёрнута в {% raw %} для Jinja.</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">установка PostgreSQL/FreeRADIUS, роли, миграции, sql-модуль</e>
    <e path="./templates/sql.j2">SQL-модуль FreeRADIUS (проверен на 3.2.5)</e>
    <e path="./templates/clients.conf.j2">RADIUS-клиенты (ingress)</e>
    <e path="./readme.md">состав, проверка, зависимости</e>
  </entrypoints>
  <depends-on>
    <d>../../../../services/db</d>
    <d>../../../terraform/modules/control-plane</d>
  </depends-on>
</dir>
