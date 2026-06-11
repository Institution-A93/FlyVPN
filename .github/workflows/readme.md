# .github/workflows

- **test.yml** — матрица `config-api`/`orchestrator`: `go build/vet/test` (интеграционные
  тесты пропускаются без `*_TEST_DSN`).
- **plan.yml** — на PR, затрагивающих `infra/terraform/**`: `tofu fmt/init/validate/plan`.
  Использует secrets → форк-PR (без секретов) пропускает.
- **deploy.yml** — на push в `main` (или `workflow_dispatch`), Environment `production`:
  `tofu apply` → собирает inventory из `tofu output` → материализует vault'ы из secrets →
  `ansible-playbook --limit egress:control`.

Список secrets/variables — в `../readme.md`.
