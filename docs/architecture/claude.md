<dir name="architecture" role="diagrams">
  <readme href="./readme.md"/>
  <purpose>визуальные схемы архитектуры (Graphviz): построенное и целевое</purpose>
  <invariants>
    <i>схема не источник истины — решения в docs/adr; схема иллюстрирует их</i>
    <i>при изменении архитектуры обновляется .dot и перерисовывается .png</i>
  </invariants>
  <entrypoints>
    <e path="./current.dot">что построено сейчас (MMVP)</e>
    <e path="./target.dot">целевая архитектура MVP (один Go-бинарь control)</e>
  </entrypoints>
  <depends-on>
    <d>../adr</d>
  </depends-on>
</dir>
