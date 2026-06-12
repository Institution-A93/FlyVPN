<project name="smart-internet" stage="mmvp">

  <readme href="./README.md"/>

  <summary>
    Умный VPN для РФ: российский трафик идёт напрямую через RU-узел, зарубежный —
    через DPI-устойчивый туннель на зарубежный узел. Доставка — .mobileconfig для
    iOS/macOS, продажа через Plati.market. Полная архитектура и тех-спек — в README.md.
  </summary>

  <doc-convention>
    <rule>В КАЖДОМ каталоге проекта лежат два файла: readme.md и claude.md.</rule>
    <rule>readme.md — markdown, человекочитаемое: что это, зачем, как запустить/проверить.</rule>
    <rule>claude.md — xml, машинно-ориентированное для агента: цель, инварианты, точки входа, зависимости.</rule>
    <rule>Проза живёт только в readme.md. claude.md НЕ дублирует прозу, а ссылается на неё через тег readme.</rule>
    <rule>При изменении назначения каталога обновляются ОБА файла.</rule>
    <claude-md-canon><![CDATA[
      <dir name="<имя>" role="<роль>">
        <readme href="./readme.md"/>
        <purpose>одна строка: за что отвечает каталог</purpose>
        <invariants>
          <i>что нельзя ломать</i>
        </invariants>
        <entrypoints>
          <e path="./...">точка входа / команда</e>
        </entrypoints>
        <depends-on>
          <d>../другой-каталог</d>
        </depends-on>
      </dir>
    ]]></claude-md-canon>
  </doc-convention>

  <decisions href="./docs/adr/readme.md">
    <d>Инфраструктура — сразу реальное облако (не local-first).</d>
    <d>Egress + control plane — Hetzner (hcloud). Ingress — Selectel.</d>
    <d>Auth — RADIUS / EAP-MSCHAPv2: auth+accounting вынесены на control plane, RU-узел без юзерских данных.</d>
    <d>ASN-split включён с самого начала: RU-префиксы напрямую, остальное в туннель.</d>
    <d>Plati-интеграция входит в MMVP (Digiseller unique-code + генерация .mobileconfig, ADR-0018).</d>
    <d>IaC — first-class для обеих ролей; узлы — cattle, ротация = tofu apply.</d>
    <d>Только OSS-компоненты в стеке. IaC-тул — OpenTofu (не Terraform: BUSL ≠ OSS).</d>
  </decisions>

  <invariants>
    <i>На RU ingress нет юзерских данных и нет секретов в репозитории.</i>
    <i>Любой узел воспроизводим из IaC + секретов оркестратора (никакой ручной настройки на узле).</i>
    <i>Инфра-стек: OpenTofu для облака, Ansible для конфигурации. Backend-сервисы — Go (ADR-0013).</i>
    <i>Только OSS-компоненты: проприетарных SaaS/софта в стеке нет (напр. GeoDNS — не Cloudflare/NS1, а OSS-вариант).</i>
    <i>Самописное минимизируем: strongSwan, FreeRADIUS, sing-box, unbound — готовые компоненты.</i>
  </invariants>

  <layout>
    <e path="./docs/adr">architecture decision records — зафиксированные решения</e>
    <e path="./infra/terraform">провижн облачных ресурсов по ролям (ingress/egress/control-plane)</e>
    <e path="./infra/ansible">конфигурация узлов</e>
    <e path="./services/config-api">выдача по коду Plati/Digiseller, генерация .mobileconfig, выдача кредов</e>
    <e path="./services/orchestrator">реестр узлов, health-check, ротация</e>
  </layout>

  <license id="AGPL-3.0-only" file="./LICENSE">
    <rule>Распространение свободно; модификации, отдаваемые как сетевой сервис, должны быть открыты (AGPL §13). См. ADR-0010.</rule>
    <rule>Открыт КОД, не боевая операционка: IP узлов, SNI-cover, анти-DPI параметры, секреты — вне репозитория.</rule>
  </license>

  <git>
    <branch>claude/kind-pasteur-2IMEz</branch>
    <rule>Разработка и пуш — только в эту ветку.</rule>
  </git>

</project>
