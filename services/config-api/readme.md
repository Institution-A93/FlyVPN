# config-api

Сервис. Точка интеграции с Plati и выдачи клиентского конфига.

Ответственность:
1. Приём вебхука Plati `/plati/issue` (HTTPS), валидация HMAC-подписи.
2. Создание юзера/подписки в PostgreSQL (см. схему в README §4).
3. Генерация EAP-кредов (username — случайная строка, password — bcrypt) и запись в `auth_credentials`.
4. Генерация `.mobileconfig` (IKEv2 + EAP-MSCHAPv2) с подстановкой значений.
5. Возврат файла в ответе Plati.

Реализация (main.go, go.mod, handlers) — TODO.
