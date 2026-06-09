# ADR-0003: Egress и control plane на Hetzner

- Статус: accepted
- Дата: 2026-06-08

## Контекст
В наличии аккаунт зарубежного хостера — Hetzner (подтверждено: git.a93.dev на
AS24940 Hetzner, FI). Нужен OpenTofu-провайдер под зарубежные узлы.

## Решение
Foreign egress и foreign control plane провижим на Hetzner Cloud через провайдер
`hcloud`.

## Последствия
Egress и control plane у одного провайдера. На phase 2 при масштабировании egress
стоит развести по разным провайдерам (как отмечено в README), чтобы не ловить
коллапс при региональных проблемах.
