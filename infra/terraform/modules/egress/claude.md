<dir name="egress" role="tf-module-egress">
  <readme href="./readme.md"/>
  <purpose>OpenTofu-модуль foreign egress на Hetzner.</purpose>
  <invariants>
    <i>На egress нет аккаунтинга и привязки к юзеру — узел не знает, кто через него ходит.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">контракт inputs/outputs</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/egress</d>
  </depends-on>
</dir>
