<project name="smart-internet" stage="mmvp">

  <readme href="./README.md"/>

  <summary>
    Умный VPN для РФ и диаспоры: трафик к «своему» региону идёт напрямую, к «чужому» —
    через DPI-устойчивый туннель. Узлы симметричны: РФ-юзер → загран-выход, диаспора →
    РФ-выход (ADR-0023). Клиент — strongSwan IKEv2; доставка профиля .mobileconfig (iOS)
    / .sswan + strongSwan-app (Android) через Telegram-бот (ADR-0022). Доступ — нативный
    FreeRADIUS/PostgreSQL; операторская поверхность — тонкий Go-сервис (панель +
    идемпотентный контракт); коммерческий учёт — самописные коннекторы за швом renew()
    (ADR-0021). Полная архитектура и тех-спек — в README.md.
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
    <d>Сплит трафика: к «своему» региону напрямую, к «чужому» в туннель; двунаправленный РФ↔загран (ADR-0023). Реализация пока GeoIP-CIDR; ASN-гранулярность — backlog B3.</d>
    <d>Plati/Digiseller (unique-code, ADR-0018) — для MMVP; в MVP платежи через самописные коннекторы за швом renew() (Platega и др., ADR-0020/0021).</d>
    <d>IaC — first-class; узлы симметричны (одна роль node: регион + вход/выход), двунаправленный РФ↔загран (ADR-0023); cattle, ротация = tofu apply.</d>
    <d>Только OSS-компоненты в стеке. IaC-тул — OpenTofu (не Terraform: BUSL ≠ OSS).</d>
    <d>Плоскость entitlement — нативный FreeRADIUS на PostgreSQL (своя auth_credentials + radacct/sqlcounter/Expiration, EAP-MSCHAPv2 из NT-hash); коммерческий учёт вне MVP (ADR-0021).</d>
    <d>Операторская поверхность — ОДИН Go-бинарь control (модульный монолит): panel (ручной CRUD юзеров + узлы) + contract (CRUD, единственный писатель) + profiles + fleet; config-api/orchestrator сливаются в него (ADR-0021).</d>
    <d>Клиентская доставка — iOS .mobileconfig / Android strongSwan .sswan, оба через Telegram-бот (ADR-0022).</d>
  </decisions>

  <invariants>
    <i>На RU ingress нет юзерских данных и нет секретов в репозитории.</i>
    <i>Любой узел воспроизводим из IaC + секретов оркестратора (никакой ручной настройки на узле).</i>
    <i>Инфра-стек: OpenTofu для облака, Ansible для конфигурации. Backend-сервисы — Go (ADR-0013).</i>
    <i>Только OSS-компоненты: проприетарных SaaS/софта в стеке нет (напр. GeoDNS — не Cloudflare/NS1, а OSS-вариант).</i>
    <i>Самописное минимизируем: strongSwan, FreeRADIUS, sing-box, unbound — готовые компоненты.</i>
    <i>Плоскость доступа — нативный FreeRADIUS на PostgreSQL; операторская поверхность — один Go-бинарь control (panel CRUD + contract + profiles + fleet), не платформа.</i>
    <i>Единственный писатель в живой entitlement — контракт renew(); платёжные коннекторы и кнопки панели ходят только через него, не в живую инфру напрямую.</i>
  </invariants>

  <layout>
    <e path="./docs/adr">architecture decision records — зафиксированные решения</e>
    <e path="./docs/architecture">визуальные схемы (Graphviz): построенное и целевое</e>
    <e path="./infra/terraform">провижн облачных ресурсов по ролям (ingress/egress/control-plane)</e>
    <e path="./infra/ansible">конфигурация узлов</e>
    <e path="(planned)/services/control" status="целевой">ЕДИНЫЙ Go-бинарь (модульный монолит): panel (ручной CRUD юзеров + узлы) + contract (CRUD над auth_credentials, единственный писатель) + profiles (.mobileconfig/.sswan) + fleet (health/реестр/автопровизия) + webhook-коннекторы (ADR-0021)</e>
    <e path="./services/config-api" status="сливается в control">генерация профилей + Issue() → пакеты profiles/contract в control</e>
    <e path="./services/orchestrator" status="сливается в control">реестр узлов, health, автопровизия → пакет fleet в control</e>
    <e path="./services/account-api" status="растворяется">самописная Go-плоскость entitlement — снимается; донор каркаса (HTTP/store/NTHash/JWT) для control (ADR-0021)</e>
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
