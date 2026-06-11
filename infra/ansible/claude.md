<dir name="ansible" role="node-configuration">
  <readme href="./readme.md"/>
  <purpose>Идемпотентная конфигурация узлов после провижна OpenTofu.</purpose>
  <invariants>
    <i>Плейбуки не содержат секретов; на MMVP — ansible-vault, на phase 2 — оркестратор (ADR-0012).</i>
    <i>Реальный inventory.ini не коммитится (IP узлов оперативно чувствительны).</i>
    <i>Прогон идемпотентен и безопасен для cattle-узла.</i>
  </invariants>
  <entrypoints>
    <e path="./site.yml">плейбук-точка входа</e>
    <e path="./roles">роли по типам узлов</e>
    <e path="./inventory.ini.example">пример inventory</e>
  </entrypoints>
  <depends-on>
    <d>../terraform</d>
  </depends-on>
</dir>
