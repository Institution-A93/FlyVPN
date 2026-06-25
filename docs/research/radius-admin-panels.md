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

**OpenWISP RADIUS** уникально закрывает оба ограничения и вдобавок всасывает кусок
будущей периферии (телефон-OTP, само-регистрация, REST API под Platega-переводчик).
Его Django-модели — это каноническая FR-схема (`radcheck`/`radacct`), поэтому
EAP-MSCHAPv2 идёт через проверенный `rlm_sql`+`NT-Password`, а управление, CoA и
счётчики — поверх тех же строк.

- **daloRADIUS** — сильный второй, если приоритет «максимально сырая прозрачность»
  и мы готовы держать RADIUS-подсистему на MariaDB; но без REST API периферию
  подключать грязнее.
- **RADIUSdesk** — слабейший фит: своя модель + MySQL + mesh-направленность.

Решение зафиксировано в **ADR-0021**. Рискованное звено (EAP-MSCHAPv2 через
`rlm_sql` на PostgreSQL) проверено спайком — см.
[spike-openwisp-eap-mschapv2.md](./spike-openwisp-eap-mschapv2.md).

## Источники

- daloRADIUS — <https://github.com/lirantal/daloradius>
- OpenWISP RADIUS — <https://github.com/openwisp/openwisp-radius/releases>,
  <https://openwisp.io/docs/dev/radius/deploy/freeradius.html>
- RADIUSdesk — <https://github.com/RADIUSdesk/rdcore>
- strongSwan eap-radius — <https://docs.strongswan.org/docs/5.9/plugins/eap-radius.html>
- (FUD-источник, вендор) — <https://cloudradius.com/is-there-a-freeradius-gui/>
