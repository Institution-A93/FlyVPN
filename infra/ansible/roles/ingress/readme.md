# role: ingress

Ansible-роль RU ingress. Настраивает узел после провижна OpenTofu-модулем `ingress`
(Selectel). Ядро «умного» продукта — сплит трафика РФ/заграница.

Состав:
- **strongSwan** — терминатор IKEv2/IPsec (UDP 500/4500). Серверный серт — **Let's Encrypt**
  (certbot HTTP-01 на `ingress_le_domain`, по умолчанию `vpn.fly-vpn.net`); EAP через
  `eap-radius` (плагин включается `load = yes`) на control plane (NT-hash/MSCHAPv2, ADR-0014).
  Пул клиентов `10.8.0.0/14` (sticky Framed-IP от RADIUS). На Debian 12 демон —
  `strongswan-starter` (charon); `swanctl.conf` грузится `swanctl --load-all`.
- **sing-box (VLESS-Reality client + TUN)** — туннель к egress и **сплит** (ADR-0015): весь
  клиентский трафик уходит в TUN; route-правила: RU `rule_set` → `direct` (выход с RU-IP),
  остальное → `vless` (туннель к egress). TUN `mtu=1320` (под двойную инкапсуляцию + LTE).
- **mesh-route.service** — `ip rule from 10.8.0.0/14 → table 100` + дефолт через TUN;
  `PartOf=sing-box` (переподнимается при рестарте sing-box, иначе маршрут в пересозданный TUN теряется).
- **ru-ruleset.timer** — ежедневная пересборка RU `rule_set` из **GeoIP-RU** (ipdeny RU zone,
  `sing-box rule-set compile`). Это список РОССИЙСКИХ подсетей (не блок-лист).

DNS клиента: пушится **публичный резолвер** (`ingress_client_dns`, 1.1.1.1); sing-box
перехватывает DNS (`sniff` + `protocol:dns → dns-out`) и резолвит через egress (не утекает
из РФ). На адрес TUN sing-box DNS не слушает — туда пушить нельзя. Kernel-NAT не нужен —
sing-box проксирует на L4 (ADR-0015).

## Маршрутизация: GeoIP-CIDR (не ASN)
Сейчас РФ/заграница определяется по совпадению dst-IP с GeoIP-RU набором. Переход на ASN —
требование backlog **B3** (точнее для операторов/банков/госуслуг).

## Известное ограничение: госсайты/банки
**Госуслуги и часть банков блокируют IP дата-центров** (в т.ч. Selectel). Их IP корректно
маршрутизируются `direct`, но назначение не отвечает нашему хостинг-адресу (проверено: TCP
не встаёт даже напрямую с ingress). Это не баг маршрутизации; чинится только резидентным
выходом — см. backlog **B4**.

## Секреты/параметры (секрет CI `INGRESS_VAULT`, ADR-0012; в репозитории нет)
Reality к egress (`ingress_egress_server`, `ingress_reality_uuid`,
`ingress_reality_public_key`, `ingress_reality_short_id`), RADIUS (`ingress_radius_server`,
`ingress_radius_secret`), `ingress_le_email`. Серт IKEv2 узел получает сам (certbot).

## Проверено (вживую, 2026-06-18)
Сквозной путь с iPhone (WiFi и LTE): IKEv2/EAP-RADIUS поднимается, заграница идёт через
egress, РФ-сайты — напрямую, DNS резолвится, MTU под LTE настроен. Не работают только
госсайты (ограничение B4, на стороне назначения).
