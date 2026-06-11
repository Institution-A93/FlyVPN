<dir name="mmvp" role="tf-env-mmvp">
  <readme href="./readme.md"/>
  <purpose>Root-модуль окружения MMVP: провайдер hcloud + вызов модулей узлов.</purpose>
  <invariants>
    <i>Поднимаются egress и control plane; ingress добавляется модулем без переделки остального.</i>
    <i>hcloud_token — sensitive, через TF_VAR_hcloud_token; в код и репозиторий не попадает.</i>
    <i>State локальный на старте, *.tfstate в .gitignore; не коммитится.</i>
  </invariants>
  <entrypoints>
    <e path="./main.tf">вызовы модулей узлов</e>
    <e path="./providers.tf">конфигурация провайдера hcloud</e>
    <e path="./variables.tf">входные переменные окружения</e>
    <e path="./terraform.tfvars.example">пример значений (без секретов)</e>
  </entrypoints>
  <depends-on>
    <d>../../modules/egress</d>
    <d>../../modules/control-plane</d>
  </depends-on>
</dir>
