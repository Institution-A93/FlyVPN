<dir name="ingress" role="ansible-role-ingress">
  <readme href="./readme.md"/>
  <purpose>Стек RU ingress: strongSwan (IKEv2/EAP-RADIUS) + sing-box (TUN + ASN-split) (ADR-0015).</purpose>
  <invariants>
    <i>На узле нет юзерских данных и RADIUS-логов; auth проксируется на control plane.</i>
    <i>ASN-split в маршрутизации sing-box: RU rule_set → direct, остальное → vless к egress.</i>
    <i>Секреты/параметры (reality, radius, серт IKEv2) только от оркестратора/vault; в репо нет — роль падает без них.</i>
    <i>DNS клиентов резолвится на egress через туннель; kernel-NAT не используется.</i>
    <i>Версия sing-box запинена; конфиг проходит sing-box check (validate).</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">установка strongSwan/sing-box, routing, rule_set</e>
    <e path="./templates/singbox-ingress.json.j2">TUN + Reality client + ASN-split</e>
    <e path="./templates/swanctl.conf.j2">IKEv2 + eap-radius</e>
    <e path="./templates/update-ru-ruleset.sh.j2">сборка RU rule_set</e>
    <e path="./readme.md">состав, секреты, статус проверки</e>
  </entrypoints>
  <depends-on>
    <d>../../../terraform/modules/ingress</d>
    <d>../control-plane</d>
    <d>../egress</d>
  </depends-on>
</dir>
