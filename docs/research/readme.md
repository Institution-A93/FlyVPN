# Research notes

Сравнения, спайки и разведка перед фиксацией решений в ADR. В отличие от ADR
(одно решение, неизменяемое), research-заметки — рабочие: они обосновывают выбор
и хранят воспроизводимые проверки, на которые ADR ссылается.

| Файл | О чём | Питает |
|------|-------|--------|
| [radius-admin-panels.md](./radius-admin-panels.md) | Сравнение OSS-панелей управления FreeRADIUS (daloRADIUS / RADIUSdesk / OpenWISP) под наш стек | ADR-0021 |
| [spike-openwisp-eap-mschapv2.md](./spike-openwisp-eap-mschapv2.md) | Воспроизводимый спайк: EAP-MSCHAPv2 из NT-hash + accounting в `radacct` на PostgreSQL 16 | ADR-0021 |
| [billing-and-bss-options.md](./billing-and-bss-options.md) | Разведка биллинг/BSS-платформ (OpenWISP/SHM/ABillS/Ubilling/Freeside/CGRateS) и почему ни одна не нужна в MVP | ADR-0021 |
