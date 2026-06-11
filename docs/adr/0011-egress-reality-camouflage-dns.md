# ADR-0011: Камуфляж egress через Reality dest; DNS на egress; proxy-режим без NAT

- Статус: accepted
- Дата: 2026-06-09

## Контекст
README §2.3/§2.4 описывает egress как «Xray Reality + nginx-fallback на 443 + NAT
исходящего трафика + unbound». При реализации роли всплыли уточнения: под VLESS-Reality
nginx на 443 не нужен и конфликтует с Xray, а NAT/MASQUERADE относится к TUN/IP-режиму,
которого в proxy-режиме нет.

## Решение
1. **Камуфляж — через Reality `dest`/`serverNames` на реальный сайт-донор.** Reality сам
   проксирует активный пробинг на dest и крадёт его TLS-хендшейк. Отдельный nginx на 443
   не разворачиваем (он конфликтует с Xray и в Reality-схеме избыточен).
2. **DNS клиентов резолвится на egress** локальным unbound (рекурсивный, слушает только
   127.0.0.1). Xray направляет DNS в unbound — резолв не утекает из РФ.
3. **Proxy-режим, не TUN.** Egress принимает VLESS-соединения и инициирует исходящие сам
   (`freedom`), поэтому MASQUERADE/NAT для egress не нужен. Приватные сети заблокированы
   routing-правилом (`geoip:private` → blackhole).
4. **Версия Xray запинена** (`egress_xray_version`) ради воспроизводимости cattle-узла.

Это уточняет/заменяет соответствующие детали README §2.3 для egress.

## Открытый вопрос (для ingress)
Транспорт mesh со стороны ingress (как «умный split» на IP/nftables передаёт трафик в
Xray): TUN-inbound vs TPROXY-redirect — решается отдельным ADR при реализации ingress.
На egress-роль выбор не влияет: сервер VLESS-Reality + freedom одинаков в обоих случаях.

## Последствия
Egress-роль проще и корректнее для Reality. Секреты Reality (private key, клиенты)
приходят от оркестратора, в репозитории отсутствуют.
