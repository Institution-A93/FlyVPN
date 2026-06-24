# .github/workflows

- **test.yml** — джоба `go`: матрица `config-api`/`orchestrator` (`go build/vet/test`,
  интеграционные тесты пропускаются без `*_TEST_DSN`); джоба `python`: бот
  (`services/bot`) — `ruff check` + смоук импортов (ADR-0019). Триггер — `services/**`.
- **plan.yml** — на PR, затрагивающих `infra/terraform/**`: `tofu fmt/init/validate/plan`.
  Использует secrets → форк-PR (без секретов) пропускает.
- **deploy.yml** — на push в `main` (или `workflow_dispatch`), Environment `production`:
  `tofu apply` → собирает inventory из `tofu output` → материализует vault'ы из secrets
  (`EGRESS_VAULT`, `CONTROL_VAULT`, опц. `CONTROL_DIGISELLER` → `zz_digiseller.yml`,
  опц. `CONTROL_BOT` → `zz_bot.yml`) → `ansible-playbook --limit egress:control`.

Список secrets/variables — в `../readme.md`.
