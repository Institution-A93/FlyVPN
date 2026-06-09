<dir name="ingress" role="tf-module-ingress">
  <readme href="./readme.md"/>
  <purpose>OpenTofu-модуль RU ingress на Selectel.</purpose>
  <invariants>
    <i>Открыты только нужные порты IKEv2; никакого юзерского состояния на диске вне FDE.</i>
  </invariants>
  <entrypoints>
    <e path="./readme.md">контракт inputs/outputs</e>
  </entrypoints>
  <depends-on>
    <d>../../../ansible/roles/ingress</d>
  </depends-on>
</dir>
