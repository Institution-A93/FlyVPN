<dir name="roles" role="ansible-roles">
  <readme href="./readme.md"/>
  <purpose>Роли конфигурации по типам узлов.</purpose>
  <invariants>
    <i>Роль отвечает ровно за стек своей роли узла.</i>
  </invariants>
  <entrypoints>
    <e path="./ingress">стек ingress</e>
    <e path="./egress">стек egress</e>
    <e path="./control-plane">стек control plane</e>
  </entrypoints>
  <depends-on/>
</dir>
