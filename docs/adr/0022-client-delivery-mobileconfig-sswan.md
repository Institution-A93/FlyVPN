# ADR-0022: Клиентская доставка — iOS `.mobileconfig` / Android strongSwan `.sswan` / Telegram

- Статус: proposed
- Дата: 2026-06-25
- Связи: переутверждает ADR-0005, ADR-0014 · следствие выбора Stack A (IKEv2-клиент)

## Контекст

Клиент→ingress — strongSwan IKEv2 + EAP-MSCHAPv2 (collateral-freedom: корп-легаси не
рубят оптом, плюс нативный клиент). Доставка профиля разная по платформам, и важный
факт: **«без app, файлом» на Android невозможно** — sing-box не умеет IKEv2; браузер
и Telegram не могут вызвать `VpnManager.provisionVpnProfile` (API package-scoped +
требует consent, у веба VPN-API нет вовсе).

## Решение

- **iOS / macOS:** `.mobileconfig` (Apple config profile) → нативный device-wide
  IKEv2, **без app**, один тап. Генерит `config-api`.
- **Android:** официальное приложение **strongSwan VPN Client** (OSS, Play/F-Droid) +
  импорт **`.sswan`**-профиля (с 1.8.0): device-wide через `VpnService`, EAP-MSCHAPv2
  поддержан. Профиль генерит `config-api`.
- **Канал доставки — Telegram-бот**: шлёт профиль под платформу (файл/ссылка/QR). Бот —
  **доставка, не VPN-runtime**.
- **Android «без app» — недостижимо**: системные Настройки — только ручной ввод (нет
  импорта файла); нативный туннель провижится только приложением через `VpnManager`
  (consent + package-scoped) — браузер/Telegram не могут. Поэтому для MVP — strongSwan-app.

**Фаза 2 (опция):** собственное Android-приложение через
`VpnManager.provisionVpnProfile(Ikev2VpnProfile)` — нативный туннель без фона,
бесшовно «как iOS».

## Последствия

`config-api` генерит и `.mobileconfig`, и `.sswan`; Telegram-бот доставляет; auth для
обеих платформ — FreeRADIUS EAP-MSCHAPv2. «Без app» — привилегия только iOS; Android
платит разовой бесплатной установкой strongSwan.
