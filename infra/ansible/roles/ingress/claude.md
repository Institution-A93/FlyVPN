<dir name="ingress" role="ansible-role-ingress">
  <readme href="./readme.md"/>
  <purpose>Стек RU ingress: strongSwan (IKEv2/EAP-RADIUS, серт Let's Encrypt) + sing-box (TUN + сплит РФ/заграница) (ADR-0015).</purpose>
  <invariants>
    <i>На узле нет юзерских данных и RADIUS-логов; auth проксируется на control plane.</i>
    <i>Сплит в sing-box: RU rule_set (GeoIP-RU CIDR) → direct, остальное → vless к egress. ASN — будущее (backlog B3).</i>
    <i>Reality/RADIUS — из секрета INGRESS_VAULT (в репо нет); серт IKEv2 — Let's Encrypt (certbot HTTP-01) на узле.</i>
    <i>DNS клиенту пушится публичный резолвер (1.1.1.1); sing-box перехватывает DNS и резолвит через egress. На адрес TUN DNS не слушает.</i>
    <i>strongSwan на Debian = strongswan-starter; eap-radius load=yes; charon рестартят для подхвата плагинов; swanctl.conf — через swanctl --load-all.</i>
    <i>mesh-route PartOf=sing-box (рестарт пересоздаёт TUN → маршрут восстанавливается); TUN mtu=1320 (LTE/двойная инкапсуляция).</i>
    <i>Госсайты/банки (госуслуги) блокируют IP дата-центров — не чинится маршрутизацией (backlog B4); ядро (заграница+РФ) работает.</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">установка strongSwan/sing-box, серт LE, routing, rule_set, swanctl load</e>
    <e path="./templates/singbox-ingress.json.j2">TUN (mtu 1320) + Reality client + сплит + DNS-hijack</e>
    <e path="./templates/swanctl.conf.j2">IKEv2 + eap-radius; пул + client DNS</e>
    <e path="./templates/eap-radius.conf.j2">плагин eap-radius (load=yes) → RADIUS на control</e>
    <e path="./templates/update-ru-ruleset.sh.j2">сборка RU rule_set из GeoIP-RU</e>
    <e path="./readme.md">состав, секреты, статус, ограничение B4</e>
  </entrypoints>
  <depends-on>
    <d>../../../terraform/modules/ingress</d>
    <d>../control-plane</d>
    <d>../egress</d>
  </depends-on>
</dir>
