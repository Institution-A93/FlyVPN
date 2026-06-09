<dir name="terraform" role="cloud-provisioning">
  <readme href="./readme.md"/>
  <purpose>Создание облачных ресурсов под узлы через OpenTofu.</purpose>
  <invariants>
    <i>Модуль организован по роли, провайдер — входная переменная.</i>
    <i>State хранится в окружении, а не в модулях; модули чистые и переиспользуемые.</i>
  </invariants>
  <entrypoints>
    <e path="./modules">модули по ролям</e>
    <e path="./environments/mmvp">окружение MMVP</e>
  </entrypoints>
  <depends-on>
    <d>./modules</d>
  </depends-on>
</dir>
