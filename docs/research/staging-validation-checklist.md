# Чек-лист валидации на узле (staging/prod)

**Дата:** 2026-06-25 · Для проверки MVP-пивота (ADR-0021/0022/0023) на **реальных
узлах** после `tofu apply` + `ansible-playbook`. Локально (PG16+FreeRADIUS 3.2.5)
проверено: движок `sql.j2` (анти-локаут, 2 пула, accounting), бинарь `control`,
миграции, рендеры конфигов, формат CoA. Ниже — то, что валидируется **только на узле**
(ядерный IPsec, живой туннель, CoA→DAE, sing-box routing).

Легенда: `[CP]` — на control-plane узле; `[IN]` — на ingress; `[CL]` — на клиенте.

## 0. Развёртывание
- [ ] `tofu apply` (egress/control) и `mmvp-ingress`; `ansible-playbook site.yml`.
- [ ] `[CP]` `systemctl status control freeradius postgresql` — все active.
- [ ] `[CP]` миграции: `psql -d smartinternet -c "\dt"` → есть `auth_credentials, radacct,
  nodes, users`; НЕТ `subscriptions, usage_log, payments` (0007 применилась).

## 1. Движок: выдача и состояния
- [ ] `[CP]` панель: `curl -s 127.0.0.1:8080/healthz` → `ok`. Зайти в `/panel` (operator-токен).
- [ ] Provision юзера (панель или `POST /api/v1/users/{id}/provision {plan}`) → получить
  `.mobileconfig` + creds; `framed_ip` в `10.8.0.0/14`.
- [ ] `[IN]` `radtest -t mschap <user> <pass> <CP_IP> 0 <radius_secret>` → **Access-Accept**
  + `Framed-IP-Address` из `10.8.0.0/14`.

## 2. Анти-локаут (главное)
- [ ] Сделать юзера истёкшим: `psql -c "UPDATE auth_credentials SET expires_at=now()-interval '1 day' WHERE username='<u>'"`.
- [ ] `[IN]` `radtest -t mschap …` → **Access-Accept** (НЕ reject!) + `Framed-IP` из
  **`10.12.0.0/14`** (restricted-пул). ← анти-локаут работает.
- [ ] Аналогично для over-cap: залить в `radacct` расход > `traffic_cap_bytes`.
- [ ] Revoke: `UPDATE … SET revoked_at=now()` → `radtest` → **Access-Reject** (туннеля нет).

## 3. Учёт (accounting → radacct)
- [ ] `[CL]` поднять туннель (iOS `.mobileconfig` / Android strongSwan `.sswan`), прогнать трафик.
- [ ] `[CP]` `psql -c "SELECT username, acctinputoctets, acctoutputoctets, acctstoptime FROM radacct WHERE username='<u>'"` → строка растёт; interim не двоится.

## 4. Walled garden (sing-box routing)
- [ ] `[CL]` (активный, `10.8`-пул): обычный сайт + Telegram доступны.
- [ ] Перевести юзера в lapsed → переподнять туннель (теперь `10.12`-пул).
- [ ] `[CL]`: **Telegram доступен**, произвольный сайт — **нет** (block). DNS резолвит.
- [ ] `[IN]` `ls -la /var/lib/sing-box/allowlist.srs` (таймер собрал из Telegram CIDR).

## 5. CoA / DAE (обрыв живой сессии)
- [ ] `[CL]` активная сессия (в `10.8`). Исчерпать кап посреди сессии (залить radacct > cap).
- [ ] `[CP]` инициировать обрыв: revoke/renew через панель **или**
  `printf "User-Name=<u>\n" | radclient <IN_IP>:3799 disconnect <dae_secret>`.
- [ ] `[IN]` `swanctl --list-sas` → SA юзера **исчезла**; `[CL]` always-on реконнект →
  теперь `Framed-IP` из `10.12` (walled garden). ← CoA замыкает анти-локаут.
- [ ] Проверить firewall: UDP `3799` открыт CP→IN (иначе Disconnect не дойдёт).

## 6. Симметрия (диаспора → РФ-выход)
- [ ] Развернуть `ingress` в загран-регионе (`ingress_self_ruleset` = локальный набор) +
  `egress` в РФ. `[CL]` за рубежом: РФ-ресурс доступен через РФ-выход; локальный трафик
  — direct.

## 7. Платежи (коннектор)
- [ ] `POST /webhooks/pay {secret, user_id, plan}` → `204`; подписка продлена
  (панель: срок сдвинут, период обнулён, nt-hash НЕ изменился).

## Откат при провале
- Доступ полностью лежит → `systemctl status control freeradius`; `journalctl -u freeradius`.
- Reject вместо walled-garden → проверить `authorize_reply_query` (пул) и включённый `mods-enabled/sql`.
- Accounting не пишется в radacct → стоковый `sites-enabled/default` должен звать `sql` в `accounting{}` (acct на 1813 — его листенер).
- CoA не рвёт → firewall 3799, `ingress_dae_secret` == `cp_control_dae_secret`.
