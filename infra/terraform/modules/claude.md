<dir name="modules" role="terraform-modules">
  <readme href="./readme.md"/>
  <purpose>Чистые переиспользуемые модули по ролям узлов.</purpose>
  <invariants>
    <i>Модуль не содержит backend/state и хардкода окружения.</i>
    <i>Outputs модуля достаточны для регистрации узла в оркестраторе.</i>
  </invariants>
  <entrypoints>
    <e path="./ingress">Selectel ingress</e>
    <e path="./egress">Hetzner egress</e>
    <e path="./control-plane">Hetzner control plane</e>
  </entrypoints>
  <depends-on/>
</dir>
