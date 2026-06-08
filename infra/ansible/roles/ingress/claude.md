<dir name="ingress" role="ansible-role-ingress">
  <readme href="./readme.md"/>
  <purpose>Стек RU ingress: strongSwan, RADIUS-клиент, nftables/ipset, Xray-client.</purpose>
  <invariants>
    <i>Никаких юзерских данных и RADIUS-логов на узле.</i>
    <i>ASN-split: daddr in @ru_prefixes → напрямую, иначе → туннель.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">состав стека</e>
  </entrypoints>
  <depends-on>
    <d>../control-plane</d>
    <d>../egress</d>
  </depends-on>
</dir>
