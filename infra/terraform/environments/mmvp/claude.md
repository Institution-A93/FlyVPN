<dir name="mmvp" role="tf-env-mmvp">
  <readme href="./readme.md"/>
  <purpose>Root-модуль окружения MMVP: один узел на роль.</purpose>
  <invariants>
    <i>До появления RU-аккаунта применяется без ingress; ingress добавляется позже без переделки egress/control-plane.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">состав окружения</e>
  </entrypoints>
  <depends-on>
    <d>../../modules</d>
  </depends-on>
</dir>
