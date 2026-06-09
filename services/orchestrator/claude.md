<dir name="orchestrator" role="service-orchestrator">
  <readme href="./readme.md"/>
  <purpose>Реестр узлов, health-check, выдача секретов, ротация.</purpose>
  <invariants>
    <i>Узлы — cattle: ротация выполняется пересозданием через IaC.</i>
    <i>Секреты выдаются узлу в рантайме и не персистятся на узле.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">ответственность и flow</e>
  </entrypoints>
  <depends-on>
    <d>../../infra/terraform</d>
  </depends-on>
</dir>
