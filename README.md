# Smart Internet — Инфраструктура MVP

Тех-спек для разработки. Описывает архитектуру, компоненты, схему данных и сценарий доставки клиентского конфига. Цель MVP — продукт, который продаётся через Plati.market, доставляется как `.mobileconfig` для устройств Apple (iOS + macOS), работает без приложения, и обеспечивает «умный» доступ к интернету: российский трафик идёт напрямую через российский узел (быстро, без двойного хопа), зарубежный — выходит через зарубежный узел через DPI-устойчивый канал.

**Скоуп платформ на MVP: только iOS и macOS** (один и тот же `.mobileconfig`-файл устанавливается на оба). Android и Windows — вне MVP, расширение в phase 2 (см. §6). Linux — не планируется на ближайшие фазы.

## 1. Архитектура в одном абзаце

Юзер на iPhone устанавливает `.mobileconfig` (профиль конфигурации iOS), который настраивает IKEv2/IPsec VPN. iPhone подключается к ближайшему серверу-входу внутри РФ (RU ingress). Ingress принимает IKEv2-сессию, авторизует пользователя через RADIUS на центральном сервере (foreign control plane). После авторизации каждый пакет от клиента маршрутизируется по destination IP: если адрес принадлежит российскому ASN — пакет уходит в интернет напрямую с RU ingress; если зарубежному — пакет уходит в туннель VLESS-Reality, ведущий к зарубежному узлу-выходу (foreign egress), а оттуда в интернет. Возвратный трафик идёт по тому же пути в обратную сторону. Все компоненты координируются центральным оркестратором, живущим за пределами РФ.

## 2. Компоненты

### 2.1 Клиент (iOS / macOS)

Без отдельного приложения. На устройство устанавливается `.mobileconfig` — стандартный Apple Configuration Profile, поддерживаемый и iOS, и macOS. Один и тот же файл работает на обеих платформах:
- На iPhone/iPad: тап по файлу → iOS просит подтвердить установку в Settings → General → VPN & Device Management → два тапа
- На Mac: double-click → System Settings → Profiles → разрешить установку

В профиле прописаны:

- IKEv2 VPN с remote-адресом `vpn.X.com` (имя домена резолвится через GeoDNS, см. §3)
- Серверный сертификат для валидации (общий для всех RU ingress)
- EAP-MSCHAPv2 с уникальным username/password для конкретного юзера (либо EAP-TLS с клиентским сертом — выбираем на этапе разработки)
- DNS-сервер внутри туннеля — внутренний IP, который маршрутизируется на foreign egress
- `OnDemandRules` для авто-подключения: всегда включён, кроме известных доверенных Wi-Fi (опционально на втором этапе)

Скриншоты-инструкции для iOS и macOS отправляются в письме после оплаты от Plati (это два разных набора скриншотов, но один и тот же файл).

### 2.2 RU Ingress

Сервер в российской юрисдикции (Selectel, Reg.ru, Timeweb или подобные). На старте — один узел, на втором этапе — несколько с failover через GeoDNS.

Стек:
- **strongSwan** — терминатор IKEv2/IPsec на UDP 500/4500
- **FreeRADIUS client (или прокси)** — для EAP-аутентификации, проксирует к центральному RADIUS на foreign control plane
- **nftables** — для ASN-based маршрутизации
- **Xray-core (VLESS-Reality client)** — для туннеля к foreign egress
- **unbound** или просто DNS-форвардер — пробрасывает DNS-запросы клиентов в туннель

Логика дата-плейна:
1. Клиент подключается, IKEv2 хэндшейк, EAP-auth через RADIUS
2. Клиент получает внутренний IP из пула `10.8.0.0/14` (sticky через `Framed-IP-Address` от RADIUS)
3. Пакеты от клиента попадают под правило nftables:
   - Если `daddr in @ru_prefixes` (ipset с RU-префиксами) — `MASQUERADE` через `eth0`, выход в интернет с RU IP узла
   - Иначе — `mark 0x100`, переходит в таблицу маршрутизации `mesh`, дефолтный роут которой — `tun-vless` (туннель к foreign egress)
4. DNS-запросы клиентов (`dport 53`) форвардятся в туннель безусловно — резолв происходит на egress

Состояние на узле минимальное и эфемерное:
- Серверный приватный ключ IKEv2 — на диске, под FDE
- Сертификат сервера (публичный) — на диске
- VLESS-Reality credentials — выдаются оркестратором при старте, не персистятся
- Список RU-префиксов — обновляется крон-задачей раз в сутки из iptoasn.com и antifilter.network
- Логи strongSwan — отключены или стримятся через mesh в центральный лог-коллектор, на узле не сохраняются
- Никаких юзерских данных на узле, RADIUS живёт за границей

Полнодисковое шифрование с remote unlock: при загрузке узел запрашивает unlock-ключ у оркестратора по защищённому каналу. Если узел изъят и перезагружен в офлайне — данные на диске нечитаемы.

### 2.3 Foreign Egress

Сервер за пределами РФ, в юрисдикции с минимальным сотрудничеством с РФ (Нидерланды, Германия, ОАЭ, Турция — выбор зависит от latency до целевой аудитории и регулировки трафика). На старте — один узел.

Стек:
- **Xray-core (VLESS-Reality server)** — терминатор mesh-туннеля
- **nginx** — отдаёт «легенду» (легитимная страница) на тот же 443 порт через fallback, чтобы при попытке прямого захода на IP отвечало как реальный сайт
- **unbound** — рекурсивный DNS-резолвер для клиентов VPN
- **iptables/nftables** — NAT для исходящего трафика клиентов

Логика дата-плейна:
1. Пакет из mesh-туннеля выходит из tun-устройства
2. Если `daddr` = внутренний DNS-резолвер (`10.x.x.1`) — попадает на локальный unbound, ответ возвращается в туннель
3. Иначе — `MASQUERADE` через `eth0`, выход в интернет с публичного IP egress-узла
4. Возвратный трафик: conntrack восстанавливает связь, пакет уходит обратно в туннель

Никакого аккаунтинга на egress — счёт ведётся на ingress. Egress не знает кто конкретно сейчас через него ходит. Это упрощает компонент и снимает privacy-нагрузку.

### 2.4 Mesh (RU ingress ↔ Foreign egress)

Каждый ingress держит постоянный туннель к каждому egress. На MVP это 1:1 (один ingress, один egress). При масштабировании — full mesh либо hub-and-spoke (зависит от количества узлов).

Транспорт — **VLESS-Reality**:
- Каждый туннель использует уникальный Reality-keypair
- SNI cover (то, под что маскируется TLS-handshake) — разный для каждого туннеля, выбирается из списка популярных CDN-доменов (например, `www.lovo.ai`, `www.icloud.com`, аналогичные). На каждом egress nginx настроен на fallback к реальному target-серверу, чтобы active probing от ТСПУ видел реалистичный TLS
- Поверх Reality — стандартный VLESS-протокол, encapsulating IP-пакеты клиентов
- Heartbeat между ingress и egress: ingress раз в 10 секунд шлёт пробу, при потере 3 проб подряд переключается на резервный туннель (если есть)

Credentials туннеля выдаются оркестратором при старте каждого узла и ротируются раз в неделю автоматически.

### 2.5 Foreign Control Plane

Центральный управляющий компонент. Живёт за пределами РФ. На MVP — один сервер; на втором этапе — кластер с репликой.

Стек:
- **FreeRADIUS** — серверная сторона EAP-аутентификации. Слушает RADIUS-запросы от ingress через VPN-туннель или поверх TLS
- **PostgreSQL** — основная БД (юзеры, подписки, креды)
- **Приложение «config-api»** — API для генерации mobileconfig, обработка Plati-вебхуков, выдача credentials оркестратору
- **Приложение «orchestrator»** — управление узлами, health-checking, secret rotation, GeoDNS update
- **Prometheus + Grafana** (опционально на MVP, обязательно на phase 2) — метрики
- **Loki или просто PostgreSQL** — логи (только critical events)

Все компоненты в одном VPC, не светятся в публичный интернет за исключением:
- `config-api` (HTTPS, принимает вебхуки от Plati)
- Прометей-эндпоинт (доступен только из VPN админа)

### 2.6 Оркестратор

Сервис на foreign control plane, отвечающий за жизненный цикл узлов и состояние сети. На MVP — простое приложение с базой состояния в Postgres и набором воркеров.

Функции:

1. **Реестр узлов.** Таблица всех ingress и egress узлов: IP, регион, статус (up/down/draining), последний heartbeat, версия конфига. Оркестратор знает что и где работает.

2. **Health checking.** Каждые 30 секунд оркестратор пробует каждый узел:
   - Ingress: проверка что IKEv2-порт отвечает (отправка фейкового IKE_SA_INIT и ожидание корректного отклика)
   - Egress: проверка что Reality-эндпоинт принимает TLS-соединения корректно
   - Mesh: проверка что трафик из ingress через mesh проходит до egress (тест-пакет через туннель)
   
   При трёх подряд неудачах — узел помечается `down`, GeoDNS обновляется.

3. **GeoDNS update.** При изменении статуса узла оркестратор пушит обновление в DNS-провайдер через API. TTL DNS-записи — 60 секунд. _Под принцип «только OSS» (ADR-0009) проприетарные Cloudflare Load Balancing / NS1 не используем — на phase 2 берём OSS-вариант (напр. PowerDNS + health-checks оркестратора). На MMVP GeoDNS не нужен (один узел)._

4. **Provisioning.** OpenTofu-модули для поднятия новых узлов. Оркестратор может вызвать `tofu apply` (через CI/CD pipeline или напрямую через cloud-провайдер API) для увеличения количества ingress в момент когда нужно. На MVP — ручной запуск; автоматизация на phase 2.

5. **Secret rotation.** Раз в неделю генерирует новые Reality keys, пушит их на соответствующую пару узлов через защищённый канал (ssh + ansible-playbook). На MVP — раз в месяц вручную; автоматизация на phase 2.

6. **Cert management.** Серверный TLS-сертификат для `vpn.X.com` (используется IKEv2 для валидации). Выписан через Let's Encrypt на foreign control plane, при обновлении (раз в 60 дней) копируется на все ingress через ansible. На MVP — ручной запуск раз в 2 месяца; автоматизация на phase 2.

7. **FDE unlock service.** Endpoint, который при загрузке нового узла отдаёт ключ расшифровки диска — после проверки идентичности узла по machine-id и подписи. На phase 2.

8. **Alerting.** Telegram-бот или email на критические события: узел упал, mesh-туннель порвался, RADIUS не отвечает, диск >90% полный, аномальный рост трафика.

### 2.7 Plati.market integration

`config-api` принимает API-вебхук от Plati.market после успешной оплаты. Спецификация — по их docs (формат «уникальный товар через API»).

Сценарий:
1. Юзер покупает товар на странице продавца на Plati
2. Plati делает HTTP-запрос на наш endpoint `https://api.X.com/plati/issue?orderid=...&signature=...`
3. `config-api` валидирует подпись Plati (HMAC)
4. Создаёт юзера в БД (если новый), создаёт подписку с end_date = now() + срок товара
5. Генерирует EAP credentials (username = случайная строка из 16 символов, password = bcrypt от случайной 32-символьной строки)
6. Записывает credentials в `auth_credentials`
7. Генерирует `.mobileconfig`-файл с подставленными значениями (см. шаблон в §4)
8. Возвращает файл в ответ Plati — он автоматически прикладывается к покупке и отдаётся юзеру через личный кабинет на oplata.info и email от Plati

При повторной покупке (продление подписки) — тот же flow, но юзер идентифицируется по `plati_buyer_id`, к существующему юзеру добавляется новая подписка, end_date его текущих credentials продлевается.

## 3. Внешние зависимости

- **DNS** — OSS-вариант с geo-routing (напр. PowerDNS), не проприетарные Cloudflare/NS1 (см. ADR-0009). Один основной домен `X.com`, поддомен `vpn.X.com` для VPN ingress, `api.X.com` для Plati-вебхука
- **Cloud провайдеры для RU ingress** — минимум два разных хостера на старте, в перспективе 5-7. Selectel, Reg.ru, Timeweb, Beget, FirstVDS — кандидаты. Регистрация через прокладку (не на основное юрлицо)
- **Cloud для foreign egress** — Hetzner, OVH, DigitalOcean, AWS. Желательно не на тех же провайдерах что ingress (избегаем коллапса при региональных проблемах)
- **Cloud для foreign control plane** — один из вышеперечисленных, в стабильной юрисдикции (Германия / Нидерланды)
- **Plati.market** — продажа и доставка цифрового товара
- **iptoasn.com + antifilter.network** — источник списка RU-префиксов
- **Let's Encrypt** — TLS-сертификаты

## 4. Схема User Database (PostgreSQL)

```
users (
    id              UUID PRIMARY KEY,
    plati_buyer_id  TEXT UNIQUE,            -- идентификатор покупателя в Plati
    email           TEXT,                    -- из Plati, для саппорта
    created_at      TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL            -- 'active', 'blocked', 'churned'
)

subscriptions (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    plati_order_id  TEXT NOT NULL UNIQUE,
    plan            TEXT NOT NULL,           -- '30d', '90d', '365d'
    started_at      TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    amount_rub      INTEGER NOT NULL,
    status          TEXT NOT NULL            -- 'active', 'expired', 'refunded'
)

auth_credentials (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    username        TEXT NOT NULL UNIQUE,    -- EAP username
    password_hash   TEXT NOT NULL,           -- bcrypt
    framed_ip       INET NOT NULL,           -- sticky internal IP, из пула 10.8.0.0/14
    issued_at       TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,             -- null если активны
    last_used_at    TIMESTAMPTZ
)

usage_log (
    user_id         UUID NOT NULL REFERENCES users(id),
    date            DATE NOT NULL,
    bytes_in        BIGINT NOT NULL DEFAULT 0,
    bytes_out       BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
)

nodes (
    id              UUID PRIMARY KEY,
    role            TEXT NOT NULL,           -- 'ingress' | 'egress' | 'control'
    region          TEXT NOT NULL,           -- 'ru-msk', 'ru-spb', 'nl-ams', etc.
    public_ip       INET NOT NULL,
    status          TEXT NOT NULL,           -- 'up', 'down', 'draining', 'maintenance'
    last_heartbeat  TIMESTAMPTZ,
    deployed_at     TIMESTAMPTZ NOT NULL,
    config_version  INTEGER NOT NULL
)

node_secrets (
    node_id         UUID NOT NULL REFERENCES nodes(id),
    secret_type     TEXT NOT NULL,           -- 'reality_private_key', 'fde_unlock_key', 'tls_cert'
    secret_value    TEXT NOT NULL,           -- зашифровано через master key (KMS / sealed)
    issued_at       TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ,
    PRIMARY KEY (node_id, secret_type)
)
```

Юзерская идентификация по `username` в RADIUS — это случайная строка, не email и не имя. Это значит даже если RU ingress изъят и через RADIUS-логи (которых не должно быть) восстановят username — этот username не ведёт обратно к юзеру без БД на control plane.

## 5. Сценарий жизни запроса

Чтобы тех-лид мог проверить себя по этой схеме, вот как должен выглядеть путь пакета:

**Юзер открывает Instagram (заблокированный сайт):**
1. iPhone → IKEv2 → RU ingress (Москва, например)
2. ingress декриптит пакет, видит destination IP Instagram (зарубежный ASN)
3. nftables маркирует пакет, отправляет в таблицу `mesh`
4. Пакет уходит в VLESS-Reality туннель к foreign egress (Амстердам)
5. egress декриптит, MASQUERADE, выход в интернет к Instagram
6. Ответ Instagram приходит на egress, conntrack восстанавливает, обратно через mesh к ingress
7. ingress инкапсулирует обратно в IKEv2-сессию, отправляет клиенту

**Юзер открывает Сбербанк:**
1. iPhone → IKEv2 → RU ingress
2. ingress видит destination IP в `@ru_prefixes` (Сбер на российском ASN)
3. nftables: `MASQUERADE` через `eth0`
4. Пакет уходит в интернет напрямую с RU IP узла
5. Сбер видит русский IP, отвечает нормально (банковские приложения часто блокируют foreign IP — здесь всё хорошо)
6. Ответ возвращается на ingress, инкапсулируется в IKEv2, отправляется клиенту

## 6. Что не входит в MVP (Phase 2+)

Чтобы не утонуть на старте, явно выносим следующее за рамки MVP:

- Несколько ingress с GeoDNS-failover (на MVP один узел)
- Несколько egress с per-user выбором локации (на MVP один)
- Автоматическая ротация секретов (на MVP — раз в месяц вручную)
- FDE remote unlock (на MVP — обычное FDE с автологином при перезагрузке; trade-off безопасности приемлем для первых юзеров)
- Telegram-канал для саппорта и дистрибуции (на MVP — email через Plati)
- Прометей-метрики, Grafana, дашборды (на MVP — структурированные логи в файл, ручной просмотр)
- Anti-DPI обфускация на first mile (AmneziaWG как замена IKEv2) — закладываем в архитектуру как pivot, но не реализуем
- Per-domain override-листы для случаев когда ASN-роутинг не отрабатывает корректно

### 6.1 Расширение на Android (Phase 2, отдельная задача)

Android не имеет встроенного аналога `.mobileconfig`. Раскатываемся через **strongSwan VPN Client** (open-source, в Play Store, доверенный мейнтейнер). Юзер-флоу: установить strongSwan из Play Store → скачать `.sswan`-файл с нашего сервера → открыть его, strongSwan ловит файл через Android intent и импортирует → connect.

Что нужно сделать:
- Добавить генератор `.sswan`-файлов в `config-api`. Формат — документированный JSON, описание на github.com/strongswan/strongswan/wiki
- В Plati-интеграции при доставке формировать ZIP с обоими файлами (`.mobileconfig` + `.sswan`) и инструкциями для обеих платформ, либо разделить на два товара на Plati
- Тестировать на двух-трёх популярных Android-устройствах (Samsung, Xiaomi, Pixel) на актуальной версии Android и одной предыдущей
- Подготовить отдельный набор инструкций со скриншотами для Android

Оценка: 3-5 дней работы. EAP-аутентификация, шифрование, серверный сертификат — те же что для iOS, дополнительной серверной работы не требуется.

Ограничение: на Android из коробки нет аналога iOS «On Demand» для авто-включения VPN. Юзер вручную тапает «Connect» в strongSwan. Это приемлемо для phase 2; для phase 3 можно подумать о Tasker-сценариях или обёртке.

### 6.2 Расширение на Windows (Phase 2, отдельная задача)

Windows нативно поддерживает IKEv2. Доставка через персонализированный PowerShell-скрипт.

Что нужно сделать:
- Генератор `.ps1`-файлов в `config-api`. Скрипт использует cmdlet `Add-VpnConnection` с параметрами:
  ```powershell
  Add-VpnConnection -Name "Smart Internet" -ServerAddress vpn.X.com `
      -TunnelType IKEv2 -AuthenticationMethod EAP `
      -EncryptionLevel Required -RememberCredential -AllUserConnection
  Set-VpnConnectionUsernamePassword (или через credential manager)
  ```
- Юзер качает `.ps1`, кликает правой кнопкой → «Run with PowerShell», подтверждает security prompt → VPN добавлен → юзер вручную коннектится из системного трея
- Подготовить инструкцию со скриншотами, особое внимание уделить шагу с подтверждением security prompt (это самое страшное для неопытных юзеров)

Оценка: 2-3 дня работы.

Ограничение: на Windows нет автоподключения без дополнительного софта. Юзер должен подключать VPN вручную при старте сессии. Phase 2 ограничение, на phase 3 можно добавить мини-приложение в трее.

### 6.3 Что значит расширение для бизнес-стороны

После добавления Android и Windows охват увеличивается с ~25-30% мобильных юзеров РФ (iOS) до ~95% устройств (iOS + macOS + Android + Windows). Это значимое расширение TAM, но **только после того как iOS-флоу отлажен на 500-1000 юзерах** — иначе разработка для трёх платформ параллельно расфокусирует команду.


## 7. Стек и оценка трудоёмкости

Язык backend-сервисов (config-api, orchestrator) — **открытое решение** (см. ADR-0013), не зафиксирован. Самописное минимизируем, по максимуму используем готовые компоненты (strongSwan, FreeRADIUS, Xray-core, unbound).

Provisioning: **OpenTofu** для cloud-ресурсов, **Ansible** для конфигурации серверов. Только OSS-компоненты (см. ADR-0009). На MVP можно без CI/CD — деплой через `make` с локальной машины.

Оценка работы на MVP (один сеньор full-stack):
- Foreign control plane (БД, config-api, базовый orchestrator, Plati-интеграция): 5-7 дней
- Foreign egress (конфигурация Xray-core, NAT, unbound): 1-2 дня
- RU ingress (strongSwan + RADIUS proxy + nftables + Xray client + ASN-roting): 3-5 дней
- Mobileconfig генерация и тестирование на iOS (двух актуальных версиях) и macOS (одной актуальной версии): 1-2 дня
- Интеграция с Plati.market (регистрация продавца, тестовая транзакция, отладка): 2-3 дня
- Health-checking и базовый failover (даже на одном узле): 1-2 дня

Итого: **2.5-3 недели** до момента «первая платная транзакция прошла, юзер получил конфиг, VPN работает».

Дальше 1-2 недели стабилизации на первых 20-50 юзерах из своего окружения, фиксим то что выявит реальное использование.

## 8. Ключевые риски

1. **Plati.market откажет в регистрации продавца / закроет аккаунт после старта.** Митигация: иметь Rocketr.net или аналог как backup; легенда продукта — «оптимизация сетевого подключения», а не «обход блокировок»
2. **RU ingress IP получит targeted-блок от ТСПУ.** Митигация: иметь готовый OpenTofu-модуль для быстрого поднятия нового узла на другом хостере; на MVP можем себе позволить ручное переключение; на phase 2 — автоматический failover через GeoDNS
3. **Foreign egress будет идентифицирован и заблокирован по IP/SNI.** Митигация: реалистичный SNI cover на Reality + nginx fallback с настоящим контентом; ротация Reality keys; план Б — несколько egress на разных провайдерах
4. **Утечка серверного приватного ключа IKEv2 (одного из RU ingress изъяли).** Митигация: PFS у IKEv2 защищает прошлые сессии; ротация серверного серта при инциденте; FDE снижает шанс утечки
5. **Регуляторное давление на самого юзера (рекламы VPN запрещены в РФ).** Митигация: маркетинг через word-of-mouth и Telegram-каналы, не публично; продукт позиционируется как «информационный сервис»

---

Документ — стартовая точка. Конкретные технологические выборы (RADIUS-конфиг, точная схема nftables, выбор OSS GeoDNS-провайдера) — на усмотрение тех-лида, выше описана архитектура и контракты между компонентами.

## Лицензия

Проект распространяется под **[AGPL-3.0](./LICENSE)** (см. [ADR-0010](./docs/adr/0010-license-agpl-3.0.md)). Запуск, форк и коммерческое использование свободны; модификации, отдаваемые как сетевой сервис, должны быть открыты (AGPL §13).

Открыт **код**, а не боевая операционка: IP узлов, актуальные SNI-cover домены Reality, конкретные анти-DPI параметры и любые секреты остаются вне репозитория.
