<dir name="ingress" role="tf-module-ingress">
  <readme href="./readme.md"/>
  <purpose>OpenTofu-модуль RU ingress на Selectel (через OpenStack-ресурсы).</purpose>
  <invariants>
    <i>Снаружи открыты только IKEv2 (UDP 500/4500) и SSH с admin CIDR.</i>
    <i>Провайдер openstack настраивается в окружении; selectel-проект — предусловие.</i>
    <i>Не валидировано против живого API — требует tofu validate/plan при аккаунте.</i>
  </invariants>
  <entrypoints>
    <e path="./main.tf">secgroup + инстанс + floating IP</e>
    <e path="./variables.tf">входные переменные</e>
    <e path="./outputs.tf">id/public_ip/name</e>
    <e path="./readme.md">контракт и предусловия</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/ingress</d>
  </depends-on>
</dir>
