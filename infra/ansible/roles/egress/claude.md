<dir name="egress" role="ansible-role-egress">
  <readme href="./readme.md"/>
  <purpose>Стек foreign egress: Xray VLESS-Reality server + unbound (рекурсивный DNS).</purpose>
  <invariants>
    <i>Нет аккаунтинга и привязки к юзеру.</i>
    <i>Камуфляж через Reality dest на реальный сайт; nginx на 443 не используется (ADR-0011).</i>
    <i>Секреты (reality private key, clients) только от оркестратора; в репозитории их нет — роль падает без них.</i>
    <i>unbound слушает только локально; в него ходит Xray, не внешний мир.</i>
    <i>Версия Xray запинена для воспроизводимости узла-cattle.</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">установка и настройка</e>
    <e path="./templates/xray-config.json.j2">конфиг VLESS-Reality</e>
    <e path="./defaults/main.yml">переменные роли</e>
    <e path="./readme.md">состав стека и проверка</e>
  </entrypoints>
  <depends-on>
    <d>../../../terraform/modules/egress</d>
  </depends-on>
</dir>
