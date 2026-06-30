# ADR-0020: Платежи MVP — через Platega.io (агрегатор с вебхуком), не Digiseller

- Статус: proposed (для MVP; ADR-0018/0007 — Digiseller — остаются для MMVP)
- Дата: 2026-06-23

## Контекст
MMVP принимал оплату через Plati/Digiseller по модели «уникального кода» (ADR-0018):
после оплаты Digiseller редиректит покупателя на `/plati/issue?uniquecode=...`, сервер
валидирует код и отдаёт `.mobileconfig`. Идентификация — по `email` из Digiseller;
веб-аккаунтов нет.

MVP (сайт `flynet.pro`) добавляет **аккаунты** (Telegram/телефон), личный кабинет и
покупку трафика «из кабинета». Модель Digiseller для этого не годится:
1. платёж не привязан к нашему `user_id` (только email покупателя);
2. нет серверного вебхука — выдача завязана на редирект покупателя;
3. нет управляемого набора способов оплаты под продукт.

Продуктовое решение: **на MVP перейти на Platega.io** — агрегатор с серверным
callback'ом, который позволяет вшить наш идентификатор в платёж (`payload`) и менять
права только по подтверждённому вебхуку.

## Решение (спецификация интеграции)

**База и авторизация.** `https://app.platega.io/`, обмен JSON по HTTPS. Заголовки на
каждый запрос:
- `X-MerchantId: <merchant_id>`
- `X-Secret: <api_key>`

Выдаются менеджером при подключении; хранятся в vault (ADR-0012), не в репозитории.

**Создание транзакции.** `POST /transaction/process` *(точный путь — сверить по
docs.platega.io, см. «Открытые места»)*, тело:
```jsonc
{
  "paymentMethod": 2,                 // см. таблицу способов
  "paymentDetails": { "amount": 300, "currency": "RUB" },
  "description": "FLY VPN — 10 ГБ / 1 мес",
  "returnUrl": "https://flynet.pro/cabinet?pay=ok",
  "failedUrl": "https://flynet.pro/cabinet?pay=fail",
  "payload": "<payments.id>"          // наш ключ привязки к user_id/заказу
}
```
Ответ содержит `transactionId` провайдера и URL платёжной страницы (редиректим туда
покупателя).

**Способы оплаты (`paymentMethod`)** — берём только три (решение продукта, без крипты):

| Способ | id | Назначение |
|---|----|----|
| СБП (QR) | `2` | основной для РФ |
| Карта РФ | `10` (`CardsRub`) / `11` (`CardAcquiring`) | выбрать по подключению |
| Зарубежная карта | `12` (`InternationalAcquiring`) | intl-эквайринг |
| ~~Криптовалюта~~ | ~~`13`~~ | **не подключаем** |

**Статус транзакции.** `GET` транзакции по `transactionId` → `{ status, amount, currency }`,
где `status ∈ { PENDING, CONFIRMED, CANCELED }` (точный перечень — сверить).

**Callback (вебхук).** Platega шлёт `POST` на наш `/webhooks/payments/platega`:
```jsonc
{ "id": "<transactionId>", "status": "CONFIRMED", "amount": 300, "payload": "<payments.id>" }
```
- Подлинность: проверяем заголовки `X-MerchantId`/`X-Secret` константным сравнением
  (уточнить, есть ли отдельная подпись тела — «Открытые места»).
- Отвечаем `200 OK` для подтверждения; идемпотентны к повторам.
- **Права меняем только из вебхука** (не по `returnUrl`-редиректу). Резерв — периодический
  poll статуса для «зависших» `PENDING`.

## Поток в account-api
1. `POST /api/v1/packages/{id}/purchase` (авторизован): создаём `payments(status=created)`,
   создаём транзакцию в Platega с `payload = payments.id`, сохраняем `provider_payment_id`,
   возвращаем `{ paymentUrl }`. Фронт редиректит.
2. Вебхук `CONFIRMED`: по `payload` находим `payments` и его `user_id`; идемпотентно
   создаём/продлеваем подписку (`kind=paid`, `+30 дней`, `traffic_bytes_limit=10 ГБ`,
   `current_period_start=now()`); если это первый успешный платёж приглашённого —
   начисляем рефереру `bonus_bytes += 1 ГБ`. Помечаем `payments.succeeded`.
3. `CANCELED`/`refunded` → обновляем статус; на возврате откатываем начисления.

Идемпотентность: `payments.idempotency_key` (наш) + `provider_payment_id` UNIQUE.

## Открытые места
- Точный путь создания транзакции (`/transaction/process`?) и имена полей запроса/ответа —
  сверить по docs.platega.io и на боевом merchant-аккаунте.
- Есть ли отдельная подпись тела вебхука (HMAC), помимо `X-Secret`, и её схема.
- Точное множество значений `status` и какие считаем «успехом».
- Карта РФ: `CardsRub` (10) vs `CardAcquiring` (11) — выбрать по условиям подключения.
- Процедура возвратов (refund) через API/панель.
- Сбор `email` для чеков (на стороне Platega или собираем у пользователя — см.
  backend-requirements §10).

## Следствия
- В `account-api` появляются таблица `payments` и эндпоинт `POST /webhooks/payments/platega`
  (см. `docs/backend-requirements.md` §4, §7, §11).
- ADR-0018 (Digiseller unique-code) и ADR-0007 **остаются для MMVP**, но MVP их не
  использует; в части способа приёма оплаты ADR-0020 **заменяет ADR-0018 в скоупе MVP**.
  После проверки на боевом аккаунте — перевести статус в `accepted` и при необходимости
  пометить ADR-0018 как `superseded by ADR-0020` для веб-продукта.
- Привязка платежа к `user_id` через `payload` снимает проблему «как связать покупку с
  аккаунтом», которая была у Digiseller-редиректа.
