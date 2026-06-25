# Биллинг / BSS-платформы — что смотрели и почему не взяли

**Дата:** 2026-06-25 · **Питает:** ADR-0021 · **Итог:** ничего из этого в MVP не нужно

## Зачем заметка

При выборе, чему уступить денежную сторону, перебрали готовые OSS-платформы. Чтобы
не возвращаться к ним по кругу, фиксируем кто это и почему мимо **для нашей
тривиальной задачи** (покупка → доступ на срок + кап). Если продукт когда-нибудь
дорастёт до лицевого счёта / периодики / CRM — заметка станет точкой входа.

## Кандидаты

| Платформа | Лицензия | Язык | Что даёт | Почему мимо (для MVP) |
|-----------|----------|------|----------|------------------------|
| **OpenWISP RADIUS** | GPLv3 | Python/Django | панель + REST + счётчики + CoA над FreeRADIUS | избыточен как фундамент; деньги (Subscriptions) — коммерческий модуль; для one-shot хватает `psql` |
| **SHM** (myshm) | Apache-2.0 | Perl | лицевой счёт, РФ-платежи (ЮKassa/T-bank/FreeKassa), события→external actions, кабинет | биллинг-бизнес; у нас один INSERT на платёж — не нужен |
| **ABillS** | GPLv2 | Perl | интегрированный ISP-BSS: биллинг+RADIUS+кабинет+РФ-платежи | WISP-монолит (PPPoE/VoIP/CRM), 90% лишнего, конфликт с приватностью (decision #10) |
| **Ubilling** (Stargazer) | GPL-2.0 | PHP | то же, CIS-направленность | то же |
| **Freeside** | AGPLv3 | Perl | зрелый ISP/VoIP BSS, RADIUS+CRM+self-service | тяжёлый, западные платёжные шлюзы (РФ из коробки нет) |
| **CGRateS** | OSS (Go) | Go | carrier-grade real-time charging, нативный RADIUS-agent, balance-unit «Internet Traffic» | движок без кабинета (фронт пишем); нужен только для real-time metered-кошелька |
| OpenWISP Subscriptions | коммерческая | — | платежи/карты для OpenWISP | платный, не OSS |
| daloRADIUS | GPL | PHP | панель + invoice-движок | PostgreSQL не для прода; нет REST; не лицевой счёт |

## Вывод (см. ADR-0021)

Задача тривиальна → плоскость доступа уже есть нативно (FreeRADIUS + PostgreSQL,
спайк-проверено), а billing = **самописные коннекторы за швом `renew()`** над своей
`auth_credentials`. Любая
платформа выше — это машинерия под бизнес крупнее нашего. Берём её **только** если
появится лицевой счёт / периодика; тогда наиболее вероятные — SHM (РФ-платежи,
Apache-2.0) или интегрированный ABillS/Ubilling.

## Источники

- OpenWISP — <https://github.com/openwisp/openwisp-radius>,
  <https://openwisp.org/commercial-support/>
- SHM — <https://github.com/danuk/shm>, <https://docs.myshm.ru/docs/setup/billing/>
- ABillS — <https://sourceforge.net/projects/abills/> ·
  Ubilling — <https://github.com/nightflyza/Ubilling>
- Freeside — <https://github.com/freeside/Freeside> ·
  CGRateS — <https://github.com/cgrates/cgrates>
