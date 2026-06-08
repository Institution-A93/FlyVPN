<dir name="ansible" role="node-configuration">
  <readme href="./readme.md"/>
  <purpose>Идемпотентная конфигурация узлов после провижна Terraform.</purpose>
  <invariants>
    <i>Плейбуки не содержат секретов; секреты приходят от оркестратора в рантайме.</i>
    <i>Прогон идемпотентен и безопасен для cattle-узла.</i>
  </invariants>
  <entrypoints>
    <e path="./roles">роли по типам узлов</e>
  </entrypoints>
  <depends-on>
    <d>../terraform</d>
  </depends-on>
</dir>
