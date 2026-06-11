<dir name="common" role="ansible-role-common">
  <readme href="./readme.md"/>
  <purpose>Базовый узел: SSH-харданинг (только ключи) + fail2ban (ADR-0016).</purpose>
  <invariants>
    <i>PasswordAuthentication no; PermitRootLogin prohibit-password.</i>
    <i>fail2ban jail sshd включён.</i>
    <i>Применяется ко всем группам узлов до ролевой настройки.</i>
  </invariants>
  <entrypoints>
    <e path="./tasks/main.yml">харданинг + fail2ban</e>
    <e path="./readme.md">состав и обоснование</e>
  </entrypoints>
  <depends-on/>
</dir>
