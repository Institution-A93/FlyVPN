<dir name="orchestrator" role="service-orchestrator">
  <readme href="./readme.md"/>
  <purpose>Go-сервис: реестр узлов + health-checking (MMVP); ротация/секреты/GeoDNS — phase 2.</purpose>
  <invariants>
    <i>Узлы — cattle: ротация выполняется пересозданием через IaC.</i>
    <i>Регистрация идемпотентна по public_ip; узел → down после threshold подряд-неудач.</i>
    <i>egress/control — TLS-проба; ingress на MMVP без активной пробы (heartbeat).</i>
    <i>Конфиг из окружения (ORCH_*); секретов в коде нет.</i>
  </invariants>
  <entrypoints>
    <e path="./cmd/orchestrator">точка входа (main)</e>
    <e path="./internal/health">пробы + Checker</e>
    <e path="./internal/store">реестр nodes (pgx)</e>
    <e path="./internal/httpapi">/healthz, /nodes</e>
    <e path="./readme.md">ответственность, конфиг, проверка</e>
  </entrypoints>
  <depends-on>
    <d>../db</d>
    <d>../../infra/terraform</d>
  </depends-on>
</dir>
