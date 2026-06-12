# ADR-0018: Интеграция Plati — через Digiseller unique-code, не HMAC-вебхук

- Статус: accepted
- Дата: 2026-06-12

## Контекст
Изначально (README §2.7, ADR-0007) предполагался POST-вебхук от Plati с HMAC-подписью.
По факту Plati.market работает на платформе **Digiseller**, и модель «уникального товара,
выдаваемого сервером продавца» иная (источник: my.digiseller.com/inside/api_general.asp):
1. После оплаты Digiseller **редиректит покупателя** на наш URL с GET-параметром
   `uniquecode` (16 символов).
2. Продавец **валидирует код через Digiseller API**, получив токен.

## Решение (точная спецификация)
- **Токен:** `POST https://api.digiseller.com/api/apilogin` с телом
  `{seller_id, timestamp, sign}`, где `sign = SHA256(api_key + timestamp)`; токен живёт
  120 минут (кэшируем).
- **Проверка кода:** `GET /api/purchases/unique-code/{code}?token=...` →
  `{id_goods, amount, email, date_pay, unique_code_state.state}`.
- config-api: эндпоинт `GET /plati/issue?uniquecode=...` валидирует код, маппит
  `id_goods → план` (`CONFIGAPI_PLAN_BY_GOODS`, иначе `CONFIGAPI_DEFAULT_PLAN`), создаёт/
  продлевает подписку (идемпотентность по `uniquecode`) и возвращает `.mobileconfig`.
- Идентификация покупателя — по `email` из Digiseller.
- Креды Digiseller (`seller_id`, `api_key`) **опциональны**: без них сервис стартует,
  `/plati/issue` отвечает 503. Активируются из vault при появлении аккаунта продавца.

## Реализация на боевом аккаунте (2026-06-12)
- **Тип товара:** `uniqueunfixed` + `content_type=DigisellerCode` (Digiseller генерит
  уникальный код на каждую покупку; склад не нужен). Категория `71` (Software → Security →
  VPN), площадка plati (`owner=1`). Создан через API (`POST /api/product/create/uniqueunfixed`)
  — товар **5937891**. Для plati обязательны локализации `ru-RU`+`en-US` и блок `prices`
  (а не `price`).
- **Авто-выдача:** `verify_code = { auto_verify: true, verify_url:
  "https://api.fly-vpn.net/plati/issue" }`. После оплаты Digiseller редиректит покупателя
  на `verify_url?uniquecode=...`, наш сервер валидирует код и отдаёт `.mobileconfig`.
- **Доставка письмом — на стороне Digiseller:** чек и сам уникальный код покупатель
  получает на указанную при оплате почту от площадки. Свой SMTP-мейлер НЕ нужен (наш
  сервер отдаёт файл по `verify_url`, а не письмом).
- **План:** `id_goods → план` задаётся `CONFIGAPI_PLAN_BY_GOODS` (напр. `5937891:30d`).
- **API-замечания:** тип товара и картинку через API задать/сменить нельзя — только тип
  при создании; правка категории/картинки — в панели. Параметр `seller` в
  `dictionary/categories/search` — это имя площадки (`plati`), не числовой seller_id.

## Открытые места
- Семантика `unique_code_state.state` (1–5) — точное множество «валидных» статусов
  уточнить по аккаунту продавца; сейчас принимаем код, если `id_goods > 0`.
- Точные имена полей ответа возможно потребуют сверки на боевом аккаунте.

## Следствия
- Убран HMAC-примитив (`internal/plati`); добавлен `internal/digiseller` (токен + проверка кода) с тестами.
- README §2.7 и config-api readme приведены в соответствие.
- ADR-0007 (Plati в MMVP) сохраняется; уточняется механизм.
