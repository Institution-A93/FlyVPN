<dir name="research" role="decision-support">
  <readme href="./readme.md"/>
  <purpose>разведка и воспроизводимые спайки перед фиксацией решений в ADR</purpose>
  <invariants>
    <i>research-заметка не отменяет ADR; ADR ссылается на research, не наоборот по приоритету</i>
    <i>спайк должен быть воспроизводимым: команды + наблюдаемый результат, не «на словах»</i>
  </invariants>
  <entrypoints>
    <e path="./radius-admin-panels.md">сравнение OSS-панелей FreeRADIUS</e>
    <e path="./spike-openwisp-eap-mschapv2.md">спайк EAP-MSCHAPv2 + accounting на PostgreSQL</e>
  </entrypoints>
  <depends-on>
    <d>../adr</d>
  </depends-on>
</dir>
