<dir name="docs" role="project-documentation">
  <readme href="./readme.md"/>
  <purpose>Документация уровня проекта: решения (ADR) и общие материалы.</purpose>
  <invariants>
    <i>Принятые архитектурные решения фиксируются как ADR, а не теряются в чате.</i>
    <i>Незрелые идеи/открытые вопросы — в backlog.md; созрев до решения, переходят в ADR.</i>
  </invariants>
  <entrypoints>
    <e path="./adr">architecture decision records</e>
    <e path="./backlog.md">пост-MMVP backlog и открытые вопросы</e>
  </entrypoints>
  <depends-on>
    <d>../README.md</d>
  </depends-on>
</dir>
