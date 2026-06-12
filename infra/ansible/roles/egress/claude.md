<dir name="egress" role="ansible-role-egress">
  <readme href="./readme.md"/>
  <purpose>Стек foreign egress: sing-box (VLESS-Reality server); DNS резолвит сам sing-box (ADR-0015/0017).</purpose>
  <invariants>
    <i>Нет аккаунтинга и привязки к юзеру.</i>
    <i>Камуфляж через Reality dest/server_name на реальный сайт; nginx на 443 не используется (ADR-0011).</i>
    <i>Секреты (reality private key, clients) только от оркестратора; в репозитории их нет — роль падает без них.</i>
    <i>DNS клиентов резолвит sing-box через публичный резолвер (запрос из egress); unbound не используется (ADR-0017).</i>
    <i>Версия sing-box запинена; конфиг проходит sing-box check (validate).</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">установка и настройка</e>
    <e path="./templates/singbox-egress.json.j2">конфиг VLESS-Reality server</e>
    <e path="./defaults/main.yml">переменные роли</e>
    <e path="./readme.md">состав стека и проверка</e>
  </entrypoints>
  <depends-on>
    <d>../../../terraform/modules/egress</d>
  </depends-on>
</dir>
