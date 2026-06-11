<dir name="workflows" role="gha-workflows">
  <readme href="./readme.md"/>
  <purpose>Workflow-файлы CI/CD: test, plan, deploy.</purpose>
  <invariants>
    <i>deploy запускается на push в main / workflow_dispatch, Environment production.</i>
    <i>inventory строится из tofu output; ingress исключён до Selectel-фазы (--limit egress:control).</i>
  </invariants>
  <entrypoints>
    <e path="./deploy.yml">apply + Ansible</e>
    <e path="./plan.yml">tofu plan</e>
    <e path="./test.yml">go тесты</e>
  </entrypoints>
  <depends-on/>
</dir>
