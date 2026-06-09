<dir name="egress" role="tf-module-egress">
  <readme href="./readme.md"/>
  <purpose>OpenTofu-модуль foreign egress на Hetzner.</purpose>
  <invariants>
    <i>На egress нет аккаунтинга и привязки к юзеру — узел не знает, кто через него ходит.</i>
    <i>Снаружи открыты только reality_port и SSH (с admin_ssh_cidrs); провайдер настраивается в окружении, не в модуле.</i>
  </invariants>
  <entrypoints>
    <e path="./main.tf">сервер + firewall</e>
    <e path="./variables.tf">входные переменные</e>
    <e path="./outputs.tf">выходы для оркестратора/inventory</e>
    <e path="./readme.md">контракт inputs/outputs</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/egress</d>
  </depends-on>
</dir>
