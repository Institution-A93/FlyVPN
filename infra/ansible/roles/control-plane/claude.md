<dir name="control-plane" role="ansible-role-control-plane">
  <readme href="./readme.md"/>
  <purpose>Стек control plane: FreeRADIUS, PostgreSQL, config-api, orchestrator.</purpose>
  <invariants>
    <i>Публично доступен только config-api (HTTPS); БД и RADIUS — в приватной сети.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">состав стека</e>
  </entrypoints>
  <depends-on>
    <d>../../../../services/config-api</d>
    <d>../../../../services/orchestrator</d>
  </depends-on>
</dir>
