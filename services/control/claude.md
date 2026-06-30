<dir name="control" role="control-plane-binary">
  <readme href="./readme.md"/>
  <purpose>Единый Go-бинарь control plane (ADR-0021): contract (единственный писатель auth_credentials) + profiles + fleet (реестр/health/CoA) + panel + webhook-коннекторы.</purpose>
  <invariants>
    <i>contract — ЕДИНСТВЕННЫЙ писатель в auth_credentials; панель и коннекторы только через него.</i>
    <i>nt-hash стабильный: на продлении не ротируется.</i>
    <i>Срок/кап/анти-локаут (выбор пула) — в FreeRADIUS authorize, не в Go; control пишет только данные (expires_at/traffic_cap_bytes/period_start).</i>
    <i>Один pgx-пул, один HTTP-сервер (внутренний), одна health-горутина.</i>
    <i>Все эндпоинты внутренние; доставку профилей клиенту опосредует бот (ADR-0022).</i>
  </invariants>
  <entrypoints>
    <e path="./cmd/control">main: wiring + сервер + health-горутина</e>
    <e path="./internal/contract">Provision/Renew/Revoke/Get/Usage/State над auth_credentials</e>
    <e path="./internal/fleet">узлы + health + CoA (DisconnectUser → DAE)</e>
    <e path="./internal/panel">HTML-панель нетех-суппорта (/panel)</e>
    <e path="./internal/httpapi">operator REST + webhook (/api/v1, /webhooks/pay)</e>
    <e path="./internal/profiles">.mobileconfig + .sswan</e>
  </entrypoints>
  <depends-on>
    <d>../db</d>
    <d>../../infra/ansible/roles/control-plane</d>
  </depends-on>
</dir>
