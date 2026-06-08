<dir name="control-plane" role="tf-module-control-plane">
  <readme href="./readme.md"/>
  <purpose>Terraform-модуль foreign control plane на Hetzner.</purpose>
  <invariants>
    <i>Публично доступны только config-api (HTTPS) и admin-канал; остальное в приватной сети.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">контракт inputs/outputs</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/control-plane</d>
  </depends-on>
</dir>
