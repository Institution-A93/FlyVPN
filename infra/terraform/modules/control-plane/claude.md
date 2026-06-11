<dir name="control-plane" role="tf-module-control-plane">
  <readme href="./readme.md"/>
  <purpose>OpenTofu-модуль foreign control plane на Hetzner.</purpose>
  <invariants>
    <i>Публично доступны только config-api (HTTPS) и admin-канал; остальное в приватной сети.</i>
  </invariants>
  <entrypoints>
    <e path="./main.tf">сервер + firewall</e>
    <e path="./variables.tf">входные переменные</e>
    <e path="./outputs.tf">выходы для оркестратора/inventory</e>
    <e path="./readme.md">контракт inputs/outputs</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/control-plane</d>
  </depends-on>
</dir>
