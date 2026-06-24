# Спайк: EAP-MSCHAPv2 из NT-hash + accounting в `radacct` на PostgreSQL 16

**Дата:** 2026-06-24 · **Питает:** ADR-0021 · **Вердикт:** ✅ путь подтверждён

## Что проверяли (рискованное звено)

Перед переездом на OpenWISP надо было снять единственную недоказанную гипотезу:
**отдаёт ли каноническая FreeRADIUS-схема на PostgreSQL 16 аутентификацию
EAP-MSCHAPv2 из одного NT-hash (без cleartext), через `rlm_sql`+`radcheck`** — то
есть тот путь, который OpenWISP документирует как параллельный rlm_rest, и который
обязателен для нашего data plane (strongSwan IKEv2 + EAP-MSCHAPv2, ADR-0005/0014).

Живой IKEv2-туннель не нужен: strongSwan `eap-radius` просто релеит EAP-MSCHAPv2 в
RADIUS, а это точно воспроизводит `eapol_test` (из wpa_supplicant).

## Стенд

- **PostgreSQL 16.13** (Ubuntu 24.04) — наш боевой мажор.
- **FreeRADIUS 3.2.5** — версия из `infra/ansible/.../control-plane/templates/sql.j2`.
- Каноническая FR-схема `mods-config/sql/main/postgresql/schema.sql`
  (`radcheck`/`radreply`/`radacct`/`radusergroup`/...) — её Django-модели OpenWISP
  отражают один-в-один.
- Юзер `alice` в `radcheck` **только** с `NT-Password` (NT-hash пароля
  `Correct-Horse-9`), никакого cleartext — верно ADR-0014.
- FreeRADIUS `sql`-модуль: dialect=postgresql, стоковые `queries.conf` (читают
  `radcheck` для authorize, пишут `radacct` для accounting).

## Результаты

### 1. Аутентификация из NT-hash

| Тест | Команда | Ожидание | Факт |
|------|---------|----------|------|
| MS-CHAP | `radtest -t mschap alice … 1812` | Access-Accept | ✅ Access-Accept |
| **EAP-MSCHAPv2** | `eapol_test -c eap.conf -a 127.0.0.1 -p 1812 -s …` | EAP-Success | ✅ `method 26 (MSCHAPV2) selected` → `SUCCESS` |
| Негатив (неверный пароль) | тот же, пароль изменён | reject | ✅ `FAILURE` |

`eapol_test` — точная модель того, что релеит strongSwan: EAP-MSCHAPv2 завершился
успехом, проверка прошла **из одного NT-hash в `radcheck` на PostgreSQL**.

> Побочно поймали грабли (не OpenWISP, а наши): если в `radcheck` положить атрибут
> вне словаря (`Max-Monthly-Traffic`), `rlm_sql` падает на парсинге check-атрибутов
> и `NT-Password` не доезжает до `mschap` → reject. Лимиты задаём реальными
> атрибутами / счётчиком, не выдуманным именем.

### 2. Учёт трафика → `radacct` (логика `MonthlyTrafficCounter`)

Послали `radclient`'ом по одной сессии: `Start` → `Interim`(100/20 МБ) →
`Interim`(250/60 МБ) → `Stop`(300/75 МБ), все с одним `Acct-Unique-Session-Id`.

```
radacct: ОДНА строка, схлопнута к финальным кумулятивным значениям
  acctsessionid=sess-3001  in=314572800  out=78643200  closed=t
usedBytes за период = Σ(in+out) = 393216000 (375 МБ)
```

- **Interim-update'ы не задвоились** — стоковый запрос делает
  `INSERT … ON CONFLICT (AcctUniqueId) DO UPDATE`, то есть среда сама держит
  «кумулятив на сессию». Это ровно тот двойной счёт, которого мы боялись в
  самописном счётчике, — здесь его нет «из коробки».
- `SUM(acctinputoctets+acctoutputoctets)` за `date_trunc('month', now())` = то, что
  вычисляет `MonthlyTrafficCounter`.

## Границы достоверности (честно)

- Проверена **контрактная плоскость данных** (FreeRADIUS ↔ PostgreSQL ↔
  `radcheck`/`radacct`), на которую OpenWISP опирается, а не сам Django-слой
  OpenWISP (его не поднимали: Celery/Redis). Чтение тех же таблиц Django-моделями —
  протоптанный путь, риск низкий.
- **CoA/Disconnect не тестировали вживую** — в спайке нет NAS-приёмника (strongSwan).
  Поддержка заявлена: OpenWISP v1.1.0 (CoA) + strongSwan `dae{}`. Остаётся отдельным
  пунктом проверки на этапе развёртывания.

## Воспроизведение

Стенд поднимается локально: `apt-get install postgresql freeradius
freeradius-postgresql eapoltest`; загрузить каноническую схему; засеять `radcheck`
NT-hash'ем (`smbencrypt` или `openssl dgst -md4 -provider legacy`); включить
`sql`-модуль (postgresql) + acct-листенер с `virtual_server = default`; гонять
`radtest`/`eapol_test`/`radclient`.
