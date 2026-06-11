<dir name="environments" role="terraform-environments">
  <readme href="./readme.md"/>
  <purpose>Root-модули окружений: провайдеры, state-backend, состав узлов.</purpose>
  <invariants>
    <i>Секреты провайдеров — через переменные окружения/секрет-стор, не в коде.</i>
  </invariants>
  <entrypoints>
    <e path="./mmvp">окружение MMVP</e>
  </entrypoints>
  <depends-on>
    <d>../modules</d>
  </depends-on>
</dir>
