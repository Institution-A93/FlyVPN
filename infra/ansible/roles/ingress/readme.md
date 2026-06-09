# role: ingress

Ansible-роль RU ingress. Настраивает узел после провижна OpenTofu-модулем `ingress`
(Selectel). Ядро «умного» продукта — ASN-split.

Состав:
- **strongSwan** — терминатор IKEv2/IPsec (UDP 500/4500), EAP через `eap-radius` на
  control plane (NT-hash/MSCHAPv2, ADR-0014). Пул клиентов `10.8.0.0/14` (sticky
  Framed-IP от RADIUS).
- **sing-box (VLESS-Reality client + TUN)** — туннель к egress и **ASN-split** (ADR-0015):
  весь клиентский трафик уходит в TUN; route-правила: RU `rule_set` → `direct` (выход с
  RU-IP), остальное → `vless` (туннель к egress).
- **mesh-route.service** — `ip rule from 10.8.0.0/14 → table` + дефолт через TUN.
- **ru-ruleset.timer** — ежедневная пересборка RU `rule_set` из OSS-источников
  (`sing-box rule-set compile`).

DNS клиентов резолвится на egress через туннель (не утекает из РФ). Kernel-NAT не нужен —
sing-box проксирует на L4 (ADR-0015).

## Секреты/параметры (vault, ADR-0012; в репозитории нет)
Reality к egress: `ingress_egress_server`, `ingress_reality_uuid`,
`ingress_reality_public_key`, `ingress_reality_short_id`. RADIUS: `ingress_radius_server`,
`ingress_radius_secret`. Серверный серт/ключ IKEv2 кладёт оркестратор (ключ — под FDE) по
путям `ingress_server_cert_path`/`ingress_server_key_path`.

## Проверено / НЕ проверено
- Конфиг sing-box валиден (`sing-box check`, v1.10.7) — TUN + Reality client + rule_set.
- strongSwan `swanctl.conf`/`eap-radius.conf` и сквозной IKEv2-путь **не валидированы**
  офлайн (нужен живой узел + клиент). Требуют проверки при деплое.
