# Architecture Decision Records

Каждый ADR фиксирует одно решение: контекст, выбор, последствия. Решения не
переписываются задним числом — если передумали, добавляется новый ADR со статусом,
заменяющим старый.

Формат статуса: `proposed` | `accepted` | `superseded by ADR-XXXX`.

| #    | Решение                                              | Статус   |
|------|------------------------------------------------------|----------|
| 0001 | Граница MMVP                                          | accepted |
| 0002 | Реальное облако с самого старта                       | accepted |
| 0003 | Egress + control plane на Hetzner                    | accepted |
| 0004 | Ingress на Selectel                                  | accepted |
| 0005 | Auth: RADIUS / EAP-MSCHAPv2                           | accepted |
| 0006 | ASN-split роутинг с самого начала                    | accepted |
| 0007 | Plati-интеграция входит в MMVP                        | accepted |
| 0008 | Конвенция документации (readme.md + claude.md)       | accepted |
| 0009 | Только OSS-компоненты; IaC-тул — OpenTofu            | accepted |
| 0010 | Лицензия проекта — AGPL-3.0                          | accepted |
| 0011 | Камуфляж egress через Reality dest; DNS на egress    | accepted |
