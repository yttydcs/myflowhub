import type { Configuration, Definition, IdentityResult, Status } from './contracts'
import {
  connectionStateLabel,
  connectionStateTone,
  formatInterval,
  formatTimestamp,
  metricDescription,
  metricLabel,
  normalizeChannels,
  parseInterval,
  sampleReading,
  sampleStatusLabel,
  type Theme,
} from './model'

export type PageName = 'resources' | 'policy' | 'connection'
export type BusyAction = 'initializing' | 'refresh' | 'identity' | 'start' | 'stop' | 'save' | undefined

export interface MetricsViewState {
  activePage: PageName
  theme: Theme
  status: Status
  definitions: Definition[]
  selectedMetric?: string
  identity?: IdentityResult
  nodeID: string
  busy: BusyAction
  error?: string
  notice?: string
  lastRefreshUnixMS?: number
}

export interface MetricsViewHandlers {
  navigate(page: PageName): void
  refresh(): void
  toggleTheme(): void
  selectMetric(metric: string): void
  savePolicy(): void
  readIdentity(): void
  start(): void
  stop(): void
  dismissError(): void
}

export interface ConnectionFields {
  stateDirectory: string
  nodeID: string
  parentNodeID: string
  endpoint: string
  parentPublicKey: string
  permit: string
}

export interface MetricsView {
  render(state: MetricsViewState): void
  readIdentityFields(): Pick<ConnectionFields, 'stateDirectory' | 'nodeID'>
  readConnectionFields(): ConnectionFields
  readConfiguration(current: Configuration): Configuration
  clearPermit(): void
}

const icons = {
  resources: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 18V8m5 10V4m5 14v-6m5 6V6"/></svg>',
  policy: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h10m4 0h2M4 12h3m4 0h9M4 18h7m4 0h5"/><path d="M14 4v4M7 10v4m4 2v4"/></svg>',
  connection: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M8 12a4 4 0 0 1 4-4h3m1 4a4 4 0 0 1-4 4H9"/><path d="m14 5 3 3-3 3M10 19l-3-3 3-3"/></svg>',
}

const pageTitles: Record<PageName, string> = {
  resources: '资源状态',
  policy: '采集策略',
  connection: '连接与身份',
}

function requiredElement<T extends Element>(root: ParentNode, selector: string): T {
  const value = root.querySelector<T>(selector)
  if (!value) throw new Error(`Metrics UI 缺少元素 ${selector}`)
  return value
}

function setText(root: ParentNode, selector: string, value: string): void {
  requiredElement<HTMLElement>(root, selector).textContent = value
}

function createText<K extends keyof HTMLElementTagNameMap>(document: Document, tag: K, className: string, value: string): HTMLElementTagNameMap[K] {
  const element = document.createElement(tag)
  element.className = className
  element.textContent = value
  return element
}

function appendProperty(document: Document, list: HTMLDListElement, label: string, value: string): void {
  list.append(createText(document, 'dt', '', label), createText(document, 'dd', '', value))
}

export function createMetricsView(root: HTMLElement, handlers: MetricsViewHandlers): MetricsView {
  root.innerHTML = `
    <a class="skip-link" href="#content-stage">跳到主内容</a>
    <div class="app-shell">
      <header class="topbar">
        <div class="brand-lockup">
          <img src="/brand/myflowhub-symbol-v5.svg" alt="" aria-hidden="true" draggable="false">
          <strong>MyFlowHub <span>Metrics</span></strong>
        </div>
        <div class="topbar-actions">
          <button class="icon-button" id="refresh-button" type="button" title="刷新状态" aria-label="刷新状态">
            <svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M20 11a8 8 0 1 0-2.34 5.66M20 4v7h-7"/></svg>
          </button>
          <button class="icon-button" id="theme-button" type="button" aria-label="切换深色主题">
            <svg class="icon theme-moon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3a6 6 0 1 0 9 9 9 9 0 0 1-9-9Z"/></svg>
            <svg class="icon theme-sun" viewBox="0 0 24 24" aria-hidden="true" hidden><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42"/></svg>
          </button>
        </div>
      </header>

      <div class="app-body">
        <aside class="sidebar" aria-label="Metrics 导航">
          <div class="sidebar-title"><span>指标节点</span><span>v1</span></div>
          <nav class="nav-scroll">
            <div class="nav-group-label">运行</div>
            <button class="nav-item" type="button" data-view="resources">${icons.resources}<span>资源状态</span><small id="resource-count">0</small></button>
            <div class="nav-group-label">配置</div>
            <button class="nav-item" type="button" data-view="policy">${icons.policy}<span>采集策略</span><small id="policy-revision">—</small></button>
            <button class="nav-item" type="button" data-view="connection">${icons.connection}<span>连接与身份</span><small id="connection-node-id">20</small></button>
          </nav>
          <button class="sidebar-node" type="button" data-view="connection" aria-label="打开连接与身份">
            <span class="node-avatar">M</span>
            <span class="node-copy"><strong id="sidebar-node-name">Metrics · Node 20</strong><small><span class="status-dot" id="sidebar-status-dot"></span><span id="sidebar-status-text">已停止</span></small></span>
            <svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6l-.05.08h-4l-.05-.08a1.7 1.7 0 0 0-1-.6 1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1l-.08-.05v-4L4 9.9a1.7 1.7 0 0 0 .6-1 1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6l.05-.08h4L14.1 4a1.7 1.7 0 0 0 1 .6 1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.4 9c.08.4.3.75.6 1l.08.05v4L20 14.1c-.3.25-.52.6-.6.9Z"/></svg>
          </button>
        </aside>

        <section class="main-stack" aria-label="Metrics 工作区">
          <div class="content-tabs">
            <div class="content-tab"><span id="tab-icon">${icons.resources}</span><span id="tab-title">资源状态</span></div>
            <div class="content-tabs-meta" id="top-timestamp">尚未刷新</div>
          </div>
          <div class="content-stage" id="content-stage" tabindex="-1">
            <section class="page" data-page="resources">
              <header class="page-head">
                <div><h1>资源状态</h1><p>Windows 设备指标与节点运行状态。</p></div>
                <button class="button button-secondary" id="page-refresh-button" type="button">刷新</button>
              </header>
              <div class="summary-strip">
                <div class="summary-item"><span class="summary-label">连接</span><span class="summary-value state"><span class="status-dot" id="summary-status-dot"></span><span id="summary-status-text">已停止</span></span></div>
                <div class="summary-item"><span class="summary-label">父节点</span><span class="summary-value" id="summary-parent">—</span></div>
                <div class="summary-item"><span class="summary-label">链路代次</span><span class="summary-value" id="summary-generation">—</span></div>
                <div class="summary-item"><span class="summary-label">通知队列</span><span class="summary-value" id="summary-notifications">0 pending · 0 dropped</span></div>
              </div>
              <div id="runtime-warnings" class="runtime-warnings" aria-live="polite"></div>
              <div class="section-heading"><h2>设备资源</h2><span>选择一项查看采样详情</span></div>
              <section class="metrics-table" aria-label="设备资源">
                <div class="metrics-head" aria-hidden="true"><span>资源</span><span>当前值</span><span>状态</span><span>采样间隔</span><span></span></div>
                <div id="metric-rows"></div>
              </section>
            </section>

            <section class="page" data-page="policy" hidden>
              <header class="page-head">
                <div><h1>采集策略</h1><p>配置启用项、远端写入边界、采样节奏和通知频道。</p></div>
                <div class="page-actions"><span class="context-label" id="policy-page-revision">尚未运行</span><button class="button button-primary" id="save-policy-button" type="button">保存配置</button></div>
              </header>
              <form class="settings-layout" id="policy-form">
                <section class="settings-section">
                  <div class="settings-section-head"><div><h2>通知频道</h2><p>频道属于同一份版本化配置；使用逗号或换行分隔。</p></div></div>
                  <div class="field"><label for="notification-channels">频道</label><input class="input mono" id="notification-channels" autocomplete="off"><span class="field-help">保存时会去重并按字典序规范化。</span></div>
                </section>
                <section class="settings-section">
                  <div class="settings-section-head"><div><h2>设备指标</h2><p>“远端可写”只对平台声明为 controllable 的资源开放；这里不提供即时本地控制。</p></div></div>
                  <div class="policy-table">
                    <div class="policy-head"><span>指标</span><span>采集</span><span>远端可写</span><span>间隔</span></div>
                    <div id="policy-rows"></div>
                  </div>
                </section>
              </form>
            </section>

            <section class="page" data-page="connection" hidden>
              <header class="page-head">
                <div><h1>连接与身份</h1><p>Metrics 使用独立 Node identity、状态根与父节点信任。</p></div>
                <span class="status-text" id="connection-header-state"><span class="status-dot"></span><span>已停止</span></span>
              </header>
              <div class="settings-layout">
                <section class="settings-section">
                  <div class="settings-section-head"><div><h2>节点连接</h2><p>连接参数只属于 Metrics，不与 Desktop Profile 共享。</p></div></div>
                  <form class="form-grid" id="connection-form">
                    <div class="field"><label for="state-directory">状态目录</label><input class="input mono" id="state-directory" value="./state/metrics-windows" required autocomplete="off"></div>
                    <div class="field"><label for="endpoint">TCP 端点</label><input class="input mono" id="endpoint" value="127.0.0.1:7341" required autocomplete="off"></div>
                    <div class="field"><label for="node-id">节点 ID</label><input class="input mono" id="node-id" value="20" required maxlength="20" inputmode="numeric" pattern="[1-9][0-9]*" autocomplete="off"></div>
                    <div class="field"><label for="parent-id">父节点 ID</label><input class="input mono" id="parent-id" value="1" required maxlength="20" inputmode="numeric" pattern="[1-9][0-9]*" autocomplete="off"></div>
                    <div class="field wide"><label for="parent-key">父节点公钥</label><input class="input mono" id="parent-key" required autocomplete="off" placeholder="raw-base64 Ed25519 public key"><span class="field-help">公开验证材料；连接前必须填写完整 raw-base64 值。</span></div>
                    <div class="field wide"><label for="permit">一次性接入凭证（可选）</label><textarea class="textarea" id="permit" maxlength="245760" autocomplete="off" spellcheck="false" placeholder="JSON provisioning permit"></textarea><span class="field-help">只用于本次连接；成功后立即清空，不写入主题偏好、普通配置或日志。</span></div>
                  </form>
                  <div class="form-actions"><button class="button button-primary" id="start-button" type="button">连接节点</button><button class="button button-danger" id="stop-button" type="button">停止节点</button></div>
                </section>
                <section class="settings-section">
                  <div class="settings-section-head"><div><h2>本机身份</h2><p>生成或读取受保护的 Ed25519 identity；只展示公开材料。</p></div><button class="button button-secondary" id="identity-button" type="button">读取身份</button></div>
                  <pre class="identity-output" id="identity-output">尚未读取身份。</pre>
                </section>
              </div>
            </section>
          </div>
        </section>

        <aside class="inspector" aria-label="检查器">
          <div class="inspector-title"><span>检查器</span><span id="inspector-context">Variable</span></div>
          <div class="inspector-content" id="inspector-content"></div>
        </aside>
      </div>
    </div>
    <div class="error-banner" id="error-banner" role="alert" hidden><span id="error-message"></span><button type="button" id="dismiss-error" aria-label="关闭错误">×</button></div>
    <div class="toast" id="notice" role="status" aria-live="polite" hidden></div>
  `

  const document = root.ownerDocument
  const pageButtons = [...root.querySelectorAll<HTMLButtonElement>('[data-view]')]
  for (const button of pageButtons) {
    button.addEventListener('click', () => handlers.navigate(button.dataset.view as PageName))
  }
  requiredElement<HTMLButtonElement>(root, '#refresh-button').addEventListener('click', handlers.refresh)
  requiredElement<HTMLButtonElement>(root, '#page-refresh-button').addEventListener('click', handlers.refresh)
  requiredElement<HTMLButtonElement>(root, '#theme-button').addEventListener('click', handlers.toggleTheme)
  requiredElement<HTMLButtonElement>(root, '#save-policy-button').addEventListener('click', handlers.savePolicy)
  requiredElement<HTMLButtonElement>(root, '#identity-button').addEventListener('click', handlers.readIdentity)
  requiredElement<HTMLButtonElement>(root, '#start-button').addEventListener('click', handlers.start)
  requiredElement<HTMLButtonElement>(root, '#stop-button').addEventListener('click', handlers.stop)
  requiredElement<HTMLButtonElement>(root, '#dismiss-error').addEventListener('click', handlers.dismissError)

  let renderedPolicyKey = ''

  function renderWarnings(status: Status): void {
    const container = requiredElement<HTMLElement>(root, '#runtime-warnings')
    const warnings = [status.connection?.last_error, status.presenter_error, status.notifications.last_error].filter((value): value is string => Boolean(value))
    container.replaceChildren(...warnings.map(value => createText(document, 'p', 'runtime-warning', value)))
    container.hidden = warnings.length === 0
  }

  function renderSamples(state: MetricsViewState): void {
    const container = requiredElement<HTMLElement>(root, '#metric-rows')
    const focusedMetric = (document.activeElement as HTMLElement | null)?.dataset.metric
    if (state.status.samples.length === 0) {
      const empty = document.createElement('div')
      empty.className = 'empty-state'
      empty.append(
        createText(document, 'strong', '', state.status.running ? '等待第一份样本' : '节点尚未运行'),
        createText(document, 'span', '', state.status.running ? '采集器返回数据后，资源会显示在这里。' : '前往“连接与身份”启动 Metrics 节点。'),
      )
      container.replaceChildren(empty)
      return
    }

    const settings = new Map(state.status.configuration?.settings.map(setting => [setting.metric, setting]))
    const definitions = new Map(state.definitions.map(definition => [definition.name, definition]))
    const rows = state.status.samples.map(sample => {
      const button = document.createElement('button')
      button.type = 'button'
      button.className = `metric-row${sample.metric === state.selectedMetric ? ' selected' : ''}`
      button.dataset.metric = sample.metric
      button.setAttribute('aria-pressed', String(sample.metric === state.selectedMetric))
      const reading = sampleReading(sample)
      const name = document.createElement('span')
      name.className = 'metric-name'
      name.append(createText(document, 'strong', '', metricLabel(sample.metric)), createText(document, 'code', '', `metrics/${sample.metric}`))
      if (sample.error) name.append(createText(document, 'small', 'metric-error', sample.error))

      const readingCell = document.createElement('span')
      readingCell.className = 'metric-reading'
      const value = createText(document, 'span', 'metric-value', reading.value)
      if (reading.suffix) value.append(createText(document, 'small', '', reading.suffix))
      readingCell.append(value)
      if (reading.progress !== undefined) {
        const track = document.createElement('span')
        track.className = 'value-track'
        const fill = document.createElement('span')
        fill.className = 'value-fill'
        fill.style.width = `${reading.progress}%`
        track.append(fill)
        readingCell.append(track)
      } else {
        readingCell.append(document.createElement('span'))
      }

      const status = document.createElement('span')
      status.className = 'metric-status'
      status.append(createText(document, 'span', `status-dot ${sample.status}`, ''), createText(document, 'span', '', sampleStatusLabel(sample.status)))
      const intervalMS = settings.get(sample.metric)?.interval_ms ?? definitions.get(sample.metric)?.interval_ms
      const interval = createText(document, 'span', 'metric-interval', formatInterval(intervalMS))
      const chevron = createText(document, 'span', 'chevron', '›')
      chevron.setAttribute('aria-hidden', 'true')
      button.append(name, readingCell, status, interval, chevron)
      button.setAttribute('aria-label', `${metricLabel(sample.metric)}，${reading.value}${reading.suffix}，${sampleStatusLabel(sample.status)}${sample.error ? `，${sample.error}` : ''}`)
      button.addEventListener('click', () => handlers.selectMetric(sample.metric))
      return button
    })
    container.replaceChildren(...rows)
    if (focusedMetric) rows.find(row => row.dataset.metric === focusedMetric)?.focus({preventScroll: true})
  }

  function renderPolicy(state: MetricsViewState): void {
    const configuration = state.status.configuration
    const signature = state.definitions.map(item => `${item.name}:${item.controllable}`).join('|')
    const key = configuration ? `${configuration.revision}:${signature}` : 'none'
    setText(root, '#policy-page-revision', configuration ? `revision ${configuration.revision}` : '尚未运行')
    if (key === renderedPolicyKey) return
    renderedPolicyKey = key
    const rows = requiredElement<HTMLElement>(root, '#policy-rows')
    const channels = requiredElement<HTMLInputElement>(root, '#notification-channels')
    if (!configuration) {
      channels.value = ''
      rows.replaceChildren(createText(document, 'div', 'empty-state compact', '节点启动后可编辑采集策略。'))
      return
    }
    channels.value = configuration.notification_channels.join(', ')
    const definitions = new Map(state.definitions.map(definition => [definition.name, definition]))
    rows.replaceChildren(...configuration.settings.map(setting => {
      const row = document.createElement('div')
      row.className = 'policy-row'
      row.dataset.policyMetric = setting.metric
      const name = document.createElement('span')
      name.className = 'policy-name'
      name.append(createText(document, 'strong', '', metricLabel(setting.metric)), createText(document, 'code', '', setting.metric))
      const enabled = createSwitch(document, `${metricLabel(setting.metric)}采集`, 'enabled', setting.enabled, false)
      const controllable = definitions.get(setting.metric)?.controllable ?? false
      const writable = createSwitch(document, `${metricLabel(setting.metric)}远端可写`, 'writable', setting.writable, !controllable)
      const interval = document.createElement('label')
      interval.className = 'interval-wrap'
      const input = document.createElement('input')
      input.className = 'interval-input'
      input.type = 'number'
      input.min = '250'
      input.max = '86400000'
      input.step = '1'
      input.value = String(setting.interval_ms)
      input.dataset.field = 'interval'
      input.setAttribute('aria-label', `${metricLabel(setting.metric)}采样间隔（毫秒）`)
      interval.append(input, document.createTextNode('ms'))
      row.append(name, enabled, writable, interval)
      return row
    }))
  }

  function renderInspector(state: MetricsViewState): void {
    const container = requiredElement<HTMLElement>(root, '#inspector-content')
    const context = requiredElement<HTMLElement>(root, '#inspector-context')
    if (state.activePage === 'resources') {
      context.textContent = 'Variable'
      const sample = state.status.samples.find(item => item.metric === state.selectedMetric)
      if (!sample) {
        container.replaceChildren(createText(document, 'div', 'empty-state inspector-empty', '选择资源后查看采样详情。'))
        return
      }
      const definition = state.definitions.find(item => item.name === sample.metric)
      const setting = state.status.configuration?.settings.find(item => item.metric === sample.metric)
      const reading = sampleReading(sample)
      const heading = document.createElement('div')
      heading.className = 'inspector-heading'
      const copy = document.createElement('div')
      copy.append(createText(document, 'h2', '', metricLabel(sample.metric)), createText(document, 'code', '', `metrics/${sample.metric}`))
      const status = createText(document, 'span', `status-text ${sample.status}`, sampleStatusLabel(sample.status))
      heading.append(copy, status)
      const current = inspectorSection(document, '当前样本')
      appendProperty(document, current.list, '值', `${reading.value}${reading.suffix}`)
      appendProperty(document, current.list, '单位', sample.unit)
      appendProperty(document, current.list, '采样于', formatTimestamp(sample.sampled_at_unix_ms))
      appendProperty(document, current.list, '观察于', formatTimestamp(sample.observed_at_unix_ms))
      if (sample.error) appendProperty(document, current.list, '错误', sample.error)
      const resource = inspectorSection(document, '资源描述')
      appendProperty(document, resource.list, 'Schema', 'mfh.metrics.sample.v1')
      appendProperty(document, resource.list, '采样间隔', formatInterval(setting?.interval_ms ?? definition?.interval_ms))
      appendProperty(document, resource.list, '平台可控', definition?.controllable ? '是' : '否')
      appendProperty(document, resource.list, '远端可写', setting?.writable ? '是' : '否')
      appendProperty(document, resource.list, 'Owner', `Node ${state.nodeID}`)
      const note = document.createElement('section')
      note.className = 'inspector-section'
      note.append(createText(document, 'h3', '', '说明'), createText(document, 'p', 'inspector-note', metricDescription(sample.metric)))
      if (definition?.controllable) note.append(createText(document, 'p', 'inspector-note', '本地界面仅展示写入边界；即时控制由具备权限的客户端完成。'))
      container.replaceChildren(heading, current.section, resource.section, note)
      return
    }

    if (state.activePage === 'policy') {
      context.textContent = 'Policy'
      const heading = inspectorHeading(document, '采集策略', 'mfh.metrics.config.v1')
      const facts = inspectorSection(document, '当前配置')
      appendProperty(document, facts.list, 'Revision', state.status.configuration ? String(state.status.configuration.revision) : '—')
      appendProperty(document, facts.list, '平台', state.status.configuration?.platform ?? 'windows')
      appendProperty(document, facts.list, '指标', String(state.status.configuration?.settings.length ?? 0))
      appendProperty(document, facts.list, '频道', String(state.status.configuration?.notification_channels.length ?? 0))
      const note = inspectorNote(document, '版本边界', '保存使用当前 revision 进行并发校验；冲突与非法配置会明确显示，不会静默覆盖。')
      container.replaceChildren(heading, facts.section, note)
      return
    }

    context.textContent = 'Connection'
    const connection = state.status.connection
    const heading = inspectorHeading(document, '连接与身份', 'Metrics local node')
    const facts = inspectorSection(document, '当前链路')
    appendProperty(document, facts.list, '状态', connectionStateLabel(connection?.state, state.status.running))
    appendProperty(document, facts.list, 'Endpoint', connection?.endpoint ?? '—')
    appendProperty(document, facts.list, '父节点', connection ? `Node ${connection.parent_node_id}` : '—')
    appendProperty(document, facts.list, '尝试', connection ? String(connection.attempt) : '—')
    appendProperty(document, facts.list, 'Generation', connection ? String(connection.link_generation) : '—')
    appendProperty(document, facts.list, '下次重试', connection?.next_retry ?? '—')
    const note = inspectorNote(document, '产品边界', 'Metrics 的 identity、状态目录和连接生命周期保持独立，不与 Desktop Profile 共享。')
    container.replaceChildren(heading, facts.section, note)
  }

  function render(state: MetricsViewState): void {
    document.documentElement.dataset.theme = state.theme
    root.setAttribute('aria-busy', String(Boolean(state.busy)))
    const connectionState = state.status.connection?.state
    const connectionLabel = connectionStateLabel(connectionState, state.status.running)
    const connectionTone = connectionStateTone(connectionState, state.status.running)
    for (const button of pageButtons) {
      const active = button.dataset.view === state.activePage
      button.classList.toggle('active', active)
      if (active) button.setAttribute('aria-current', 'page')
      else button.removeAttribute('aria-current')
    }
    for (const page of root.querySelectorAll<HTMLElement>('[data-page]')) page.hidden = page.dataset.page !== state.activePage
    setText(root, '#tab-title', pageTitles[state.activePage])
    requiredElement<HTMLElement>(root, '#tab-icon').innerHTML = icons[state.activePage]
    setText(root, '#resource-count', String(state.status.samples.length))
    setText(root, '#policy-revision', state.status.configuration ? `rev ${state.status.configuration.revision}` : '—')
    setText(root, '#connection-node-id', state.nodeID)
    setText(root, '#sidebar-node-name', `Metrics · Node ${state.nodeID}`)
    setText(root, '#sidebar-status-text', connectionLabel)
    setText(root, '#summary-status-text', connectionLabel)
    setText(root, '#summary-parent', state.status.connection ? `Node ${state.status.connection.parent_node_id} · ${state.status.connection.endpoint}` : '—')
    setText(root, '#summary-generation', state.status.connection ? `generation ${state.status.connection.link_generation}` : '—')
    setText(root, '#summary-notifications', `${state.status.notifications.queue_depth} pending · ${state.status.notifications.dropped} dropped`)
    setText(root, '#top-timestamp', state.lastRefreshUnixMS ? `最后刷新 ${formatTimestamp(state.lastRefreshUnixMS)}` : '尚未刷新')
    for (const selector of ['#sidebar-status-dot', '#summary-status-dot']) {
      const dot = requiredElement<HTMLElement>(root, selector)
      dot.className = `status-dot ${connectionTone}`
    }
    const headerState = requiredElement<HTMLElement>(root, '#connection-header-state')
    headerState.className = `status-text ${connectionTone}`
    headerState.querySelector<HTMLElement>('.status-dot')!.className = `status-dot ${connectionTone}`
    headerState.querySelector<HTMLElement>('span:last-child')!.textContent = connectionLabel

    const busy = Boolean(state.busy)
    requiredElement<HTMLButtonElement>(root, '#refresh-button').disabled = busy
    requiredElement<HTMLButtonElement>(root, '#page-refresh-button').disabled = busy
    const save = requiredElement<HTMLButtonElement>(root, '#save-policy-button')
    save.disabled = busy || !state.status.configuration
    save.textContent = state.busy === 'save' ? '保存中…' : '保存配置'
    const identity = requiredElement<HTMLButtonElement>(root, '#identity-button')
    identity.disabled = busy
    identity.textContent = state.busy === 'identity' ? '读取中…' : '读取身份'
    const start = requiredElement<HTMLButtonElement>(root, '#start-button')
    start.disabled = busy || state.status.running
    start.textContent = state.busy === 'start' ? '连接中…' : (state.status.running ? '节点已运行' : '连接节点')
    const stop = requiredElement<HTMLButtonElement>(root, '#stop-button')
    stop.disabled = busy || !state.status.running
    stop.textContent = state.busy === 'stop' ? '停止中…' : '停止节点'
    const theme = requiredElement<HTMLButtonElement>(root, '#theme-button')
    theme.setAttribute('aria-label', state.theme === 'light' ? '切换深色主题' : '切换浅色主题')
    requiredElement<SVGElement>(theme, '.theme-moon').toggleAttribute('hidden', state.theme === 'dark')
    requiredElement<SVGElement>(theme, '.theme-sun').toggleAttribute('hidden', state.theme === 'light')

    const identityOutput = state.identity ? `Node ${state.identity.node_id}\npublic key · ${state.identity.public_key}` : '尚未读取身份。'
    setText(root, '#identity-output', identityOutput)
    const errorBanner = requiredElement<HTMLElement>(root, '#error-banner')
    errorBanner.hidden = !state.error
    setText(root, '#error-message', state.error ?? '')
    const notice = requiredElement<HTMLElement>(root, '#notice')
    notice.hidden = !state.notice
    notice.textContent = state.notice ?? ''

    renderWarnings(state.status)
    renderSamples(state)
    renderPolicy(state)
    renderInspector(state)
  }

  return {
    render,
    readIdentityFields: () => ({
      stateDirectory: requiredElement<HTMLInputElement>(root, '#state-directory').value,
      nodeID: requiredElement<HTMLInputElement>(root, '#node-id').value,
    }),
    readConnectionFields: () => {
      const form = requiredElement<HTMLFormElement>(root, '#connection-form')
      if (!form.reportValidity()) throw new Error('请先修正连接表单中的无效字段')
      return {
        stateDirectory: requiredElement<HTMLInputElement>(root, '#state-directory').value,
        nodeID: requiredElement<HTMLInputElement>(root, '#node-id').value,
        parentNodeID: requiredElement<HTMLInputElement>(root, '#parent-id').value,
        endpoint: requiredElement<HTMLInputElement>(root, '#endpoint').value,
        parentPublicKey: requiredElement<HTMLInputElement>(root, '#parent-key').value,
        permit: requiredElement<HTMLTextAreaElement>(root, '#permit').value,
      }
    },
    readConfiguration: current => {
      const form = requiredElement<HTMLFormElement>(root, '#policy-form')
      if (!form.reportValidity()) throw new Error('请先修正采集策略中的无效字段')
      const settings = [...root.querySelectorAll<HTMLElement>('[data-policy-metric]')].map(row => {
        const metric = row.dataset.policyMetric!
        return {
          metric,
          enabled: requiredElement<HTMLInputElement>(row, '[data-field="enabled"]').checked,
          writable: requiredElement<HTMLInputElement>(row, '[data-field="writable"]').checked,
          interval_ms: parseInterval(requiredElement<HTMLInputElement>(row, '[data-field="interval"]').value, metric),
        }
      })
      if (settings.length === 0) throw new Error('当前配置没有可保存的指标')
      return {
        ...current,
        settings,
        notification_channels: normalizeChannels(requiredElement<HTMLInputElement>(root, '#notification-channels').value),
      }
    },
    clearPermit: () => { requiredElement<HTMLTextAreaElement>(root, '#permit').value = '' },
  }
}

function createSwitch(document: Document, label: string, field: 'enabled' | 'writable', checked: boolean, disabled: boolean): HTMLLabelElement {
  const wrapper = document.createElement('label')
  wrapper.className = 'switch'
  const input = document.createElement('input')
  input.type = 'checkbox'
  input.checked = checked
  input.disabled = disabled
  input.dataset.field = field
  input.setAttribute('aria-label', label)
  const visual = document.createElement('span')
  visual.setAttribute('aria-hidden', 'true')
  wrapper.append(input, visual)
  return wrapper
}

function inspectorHeading(document: Document, title: string, subtitle: string): HTMLElement {
  const heading = document.createElement('div')
  heading.className = 'inspector-heading'
  const copy = document.createElement('div')
  copy.append(createText(document, 'h2', '', title), createText(document, 'code', '', subtitle))
  heading.append(copy)
  return heading
}

function inspectorSection(document: Document, title: string): {section: HTMLElement; list: HTMLDListElement} {
  const section = document.createElement('section')
  section.className = 'inspector-section'
  const list = document.createElement('dl')
  list.className = 'property-list'
  section.append(createText(document, 'h3', '', title), list)
  return {section, list}
}

function inspectorNote(document: Document, title: string, value: string): HTMLElement {
  const section = document.createElement('section')
  section.className = 'inspector-section'
  section.append(createText(document, 'h3', '', title), createText(document, 'p', 'inspector-note', value))
  return section
}
