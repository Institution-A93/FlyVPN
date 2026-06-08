<dir name="egress" role="ansible-role-egress">
  <readme href="./readme.md"/>
  <purpose>Стек foreign egress: Xray Reality-server, nginx-fallback, unbound, NAT.</purpose>
  <invariants>
    <i>Нет аккаунтинга и привязки к юзеру.</i>
    <i>Прямой заход на IP отвечает как реальный сайт (nginx fallback).</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">состав стека</e>
  </entrypoints>
  <depends-on/>
</dir>
