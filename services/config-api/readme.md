# config-api

Сервис. Точка интеграции с Plati и выдачи клиентского конфига. Язык — открытое
решение (ADR-0013); реализация ждёт выбора языка.

Ответственность:
1. Приём вебхука Plati `/plati/issue` (HTTPS), валидация HMAC-подписи.
2. Создание юзера/подписки в PostgreSQL (схема — `../db`).
3. Генерация EAP-кредов и запись в `auth_credentials`: username + NT-hash пароля
   (MD4(UTF-16LE), hex) — для MSCHAPv2, не bcrypt (ADR-0014).
4. Генерация `.mobileconfig` из шаблона с подстановкой значений.
5. Возврат файла в ответе Plati.

## Шаблон профиля
`profile.mobileconfig.tmpl` — Apple Configuration Profile для IKEv2 + EAP-MSCHAPv2,
языко-независимый (токены `{{NAME}}` подставляет сервис). Проверен на well-formed XML.

Токены:

| Токен                     | Значение |
|---------------------------|----------|
| `{{PROFILE_IDENTIFIER}}`  | reverse-DNS id профиля (напр. `com.X.vpn.<uuid>`) |
| `{{PROFILE_UUID}}`        | UUID профиля |
| `{{PAYLOAD_UUID}}`        | UUID VPN-payload |
| `{{DISPLAY_NAME}}`        | отображаемое имя (напр. «Smart Internet») |
| `{{ORG_NAME}}`            | организация |
| `{{VPN_REMOTE_ADDRESS}}`  | адрес ingress (`vpn.X.com`) |
| `{{VPN_REMOTE_IDENTIFIER}}` | идентификатор сервера (CN/домен серт.) |
| `{{SERVER_CA_CN}}`        | CN издателя серверного сертификата |
| `{{EAP_USERNAME}}`        | EAP username |
| `{{EAP_PASSWORD}}`        | EAP password |

OnDemand (авто-подключение) — phase 2, в шаблоне не задаётся.

Реализация (HTTP-сервер, handlers, store) — TODO после выбора языка.
