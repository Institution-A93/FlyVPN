<dir name="infra" role="infrastructure-as-code">
  <readme href="./readme.md"/>
  <purpose>IaC для всех узлов: создание ресурсов (Terraform) и конфигурация (Ansible).</purpose>
  <invariants>
    <i>Никакой ручной настройки узла вне IaC.</i>
    <i>Секреты не коммитятся; узел получает их от оркестратора при старте.</i>
    <i>Узлы — cattle: ротация выполняется пересозданием, а не правкой на месте.</i>
  </invariants>
  <entrypoints>
    <e path="./terraform">провижн ресурсов</e>
    <e path="./ansible">конфигурация узлов</e>
  </entrypoints>
  <depends-on>
    <d>../services/orchestrator</d>
  </depends-on>
</dir>
