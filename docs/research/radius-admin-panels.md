# OSS-панели управления FreeRADIUS — сравнение под наш стек

**Дата:** 2026-06-24 · **Питает:** ADR-0021 · **Статус:** учтено в решении

> **Итог (ADR-0021): ни одну панель не берём.** OpenWISP — оба (панель+API), но
> тяжёлый; daloRADIUS — панель без программного API + MariaDB-центр; OpenWISP
> Subscriptions — коммерческий. Коммерческий учёт вынесен из MVP (коннекторы), панель
> нужна только техническая, а идемпотентный контракт — наш в любом случае. Поэтому
> операторская поверхность = **тонкий свой Go-сервис (панель + контракт)**. Сравнение
> ниже сохранено как разведка ландшафта.

## Зачем

Мы решили уступить плоскость entitlement (учётки, квота, учёт трафика, отрезание)
RADIUS-среде, а не держать самописную Go-плоскость (см. ADR-0021). Тогда нужна
операторская веб-панель: прозрачная для DevSecOps, чтобы видеть и править ошибки
руками. Вопрос — какая.

## Кандидаты (живые, OSS)

| Проект | Последний релиз | Стек | ДНК |
|--------|-----------------|------|-----|
| **daloRADIUS** | 2.3 (май 2026) | PHP | сырой CRUD по нативной FR-схеме; ISP/hotspot |
| **RADIUSdesk** | активен (дек 2025) | CakePHP + ExtJS | своя модель `rd_cake`; WiFi/mesh/ваучеры |
| **OpenWISP RADIUS** | 1.2.2 (апр 2026) | Django/Python | REST API + Django-модели поверх FR-схемы; captive-portal |

Мимо: **RadMan** (легче, беднее), **Grafana** поверх `radacct` (только read-only
дашборды, не управление). Вендорские «daloRADIUS мёртв» — FUD от cloudradius.com,
продающего свой managed-продукт; по факту daloRADIUS жив (2.3, май 2026).

## Решает не «вообще лучше», а два НАШИХ жёстких ограничения

1. **PostgreSQL 16** — control-plane уже на нём, проверен (`services/db`, FreeRADIUS 3.2.5).
2. **strongSwan IKEv2 + EAP-MSCHAPv2 на NT-hash** (ADR-0005/0014): RADIUS обязан
   отдавать `NT-Password`, cleartext в БД нет.

| Критерий (наш) | daloRADIUS | RADIUSdesk | **OpenWISP** |
|----------------|-----------|------------|--------------|
| PostgreSQL | ⚠️ «не для прода, тестируют на MariaDB» | ⚠️ MySQL-ДНК | ✅ предпочтителен |
| EAP-MSCHAPv2 / NT-hash | ✅ `rlm_sql`+`radcheck` | ⚠️ через свою модель | ✅ `rlm_sql` поддержан параллельно rlm_rest |
| Прозрачность для DevSecOps | ✅✅ сырой SQL CRUD | ⚠️ своя модель поверх | ✅ Django-модели = FR-схема; видно и в `psql` |
| CoA / Disconnect (рычаг обрыва) | ✅ PoD/CoA | ✅ | ✅ с v1.1.0 |
| Месячная data-квота | billing-engine | ✅ data-limits | ✅✅ `MonthlyTrafficCounter` / `MonthlySubscriptionTrafficCounter` |
| REST API для периферии | ❌ только PHP-UI | частично | ✅✅ OpenAPI |
| Телефон/SMS, само-регистрация | ❌ | ваучеры | ✅✅ PhoneToken, SMS, self-registration, social login |
| Footprint / безопасность | PHP, история CVE → запирать | CakePHP+ExtJS тяжёлый | Django+Celery+Redis — больше частей, зрелая authz/2FA |

## Вывод

> **Итог (ADR-0021): панель НЕ берём — ни OpenWISP, ни другую.** Операторская
> поверхность = тонкий свой Go-бинарь `control` (panel CRUD + contract) над **своей
> `auth_credentials`** (не canonical radcheck). Сравнение ниже — разведка ландшафта:
> почему ни одна готовая панель не подошла под наш стек (PostgreSQL + EAP-MSCHAPv2 +
> один человек).

Для справки, как читался ландшафт на момент разведки:
- **OpenWISP** — единственный зрелый с панелью+API на PostgreSQL, но Django+Celery+
  Redis тяжелы под команду из трёх человек; и его модели — canonical `radcheck`,
  который нам не нужен (своя схема).
- **daloRADIUS** — панель без программного API + MariaDB-центр.
- **RADIUSdesk** — своя модель + MySQL + mesh-направленность.

Рискованное звено (EAP-MSCHAPv2 из NT-hash на PostgreSQL) проверено спайком — см.
[spike-openwisp-eap-mschapv2.md](./spike-openwisp-eap-mschapv2.md); механизм тот же
для нашей `auth_credentials`.

## Источники

- daloRADIUS — <https://github.com/lirantal/daloradius>
- OpenWISP RADIUS — <https://github.com/openwisp/openwisp-radius/releases>,
  <https://openwisp.io/docs/dev/radius/deploy/freeradius.html>
- RADIUSdesk — <https://github.com/RADIUSdesk/rdcore>
- strongSwan eap-radius — <https://docs.strongswan.org/docs/5.9/plugins/eap-radius.html>
- (FUD-источник, вендор) — <https://cloudradius.com/is-there-a-freeradius-gui/>
