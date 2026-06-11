# ADR-0015: sing-box как транспорт mesh (обе стороны); ASN-split в маршрутизации sing-box

- Статус: accepted
- Дата: 2026-06-09

## Контекст
ingress должен уводить RU-трафик напрямую, а остальное — в DPI-устойчивый Reality-туннель
к egress. Reality (Xray) — это L4-прокси, не IP-туннель; чтобы загнать IP-пакеты клиента в
туннель, нужен TUN. Xray-core нативного TUN не имеет (нужен tun2socks). sing-box имеет
нативный TUN + VLESS-Reality client/server и единый формат конфига.

## Решение
1. **sing-box по обе стороны mesh** (GPLv3, OSS — ADR-0009 соблюдён): на ingress —
   VLESS-Reality client + TUN; на egress — VLESS-Reality server. Заменяет Xray-core на
   обоих узлах (egress больше ничем к Xray не привязан — nginx-fallback убран в ADR-0011).
   Один бинарь, один формат конфига, один цикл обновлений. Версия запинена.
2. **ASN-split — в маршрутизации sing-box**, а не через nftables-ipset:
   - весь клиентский трафик (источник из пула strongSwan 10.8.0.0/14) направляется в TUN
     sing-box одним `ip rule from 10.8.0.0/14 lookup <mesh>` + дефолт-роут таблицы на TUN;
   - route-правила sing-box: `rule_set` RU-префиксов → outbound `direct` (выход с RU-IP
     ingress), иначе → outbound `vless` (туннель к egress);
   - `rule_set` собирается из тех же источников (iptoasn.com / antifilter.network) и
     обновляется кроном (`sing-box rule-set compile`).
   Это **уточняет механизм ADR-0006** (было: nftables-ipset + таблица mesh + tun-vless),
   сохраняя его решение и данные: RU — напрямую, остальное — в туннель.
3. **Без kernel-NAT/forwarding для дата-плейна**: sing-box проксирует на L4 (инициирует
   исходящие от хоста), поэтому MASQUERADE для клиентского трафика не нужен. RU-сайты видят
   RU-IP узла (direct), зарубежные — IP egress.
4. **DNS** клиентов резолвится на egress: dns-сервер sing-box с `detour: vless` (или egress
   unbound через туннель) — резолв не утекает из РФ.

## Следствия
- egress-роль переписывается с Xray на sing-box (server).
- ingress-роль: strongSwan (IKEv2 + EAP-RADIUS на control plane) + sing-box (TUN + split) +
  cron обновления rule_set. Минимум nftables.
- README §2.2–2.4 (Xray-core, nftables-NAT, tun-vless) уточняются под sing-box.
- Конфиги sing-box валидируются `sing-box check`.
