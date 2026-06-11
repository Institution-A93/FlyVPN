<dir name=".github" role="ci-cd">
  <readme href="./readme.md"/>
  <purpose>GitHub Actions: тесты, tofu plan на PR, apply+Ansible на push (ADR-0016).</purpose>
  <invariants>
    <i>Hosted-раннеры (публичный репо). Деплой — только на push в защищённую ветку + Environment с ревью.</i>
    <i>Секретов в git нет; vault'ы узлов материализуются из GitHub Secrets на раннере.</i>
    <i>Remote state — Hetzner Object Storage (s3-backend).</i>
  </invariants>
  <entrypoints>
    <e path="./workflows/deploy.yml">apply + Ansible</e>
    <e path="./workflows/plan.yml">tofu plan на PR</e>
    <e path="./workflows/test.yml">go build/vet/test</e>
    <e path="./readme.md">список secrets/variables и предусловия</e>
  </entrypoints>
  <depends-on>
    <d>../infra/terraform</d>
    <d>../infra/ansible</d>
  </depends-on>
</dir>
