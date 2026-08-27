import './style.css'
import { api, type Settings } from './api'
import { errorText, pages, pretty, type Page } from './model'

const root = document.querySelector<HTMLDivElement>('#app')!
let currentPage: Page = 'overview'
let settings: Settings
let activeSubscription = 0
let polling = false

root.innerHTML = `
  <div class="shell">
    <aside>
      <div class="brand"><span class="brand-mark">M</span><div><strong>MyFlowHub</strong><small>resource control plane</small></div></div>
      <nav>${pages.map((page) => `<button data-page="${page.id}"><span>${page.label}</span><small>${page.hint}</small></button>`).join('')}</nav>
      <div class="connection-card"><i id="status-dot"></i><div><strong id="status-label">未连接</strong><small id="status-detail">等待状态</small></div></div>
    </aside>
    <main>
      <header><div><p class="eyebrow">CANONICAL DESKTOP</p><h1 id="page-title">运行概览</h1></div><div class="header-actions"><label>目标节点 <input id="owner-id" value="1" inputmode="numeric"></label><button id="refresh" class="secondary">刷新</button></div></header>
      <div id="notice" role="status"></div>
      <section id="overview" class="page">
        <div class="hero"><div><p class="eyebrow">TREE · SUBSCRIPTION · COMMAND</p><h2>一个稳定资源入口，覆盖所有链路</h2><p>桌面端只消费资源目录、变量、流和指令。TCP、QUIC、蓝牙等链路差异停留在运行时。</p></div><div class="hero-actions"><button id="connect">连接父节点</button><button id="disconnect" class="secondary">断开</button></div></div>
        <div class="grid three"><article><span>连接状态</span><strong id="overview-state">—</strong><small id="overview-endpoint">—</small></article><article><span>节点身份</span><strong id="overview-node">—</strong><small id="overview-key">—</small></article><article><span>Hub 健康</span><strong id="overview-health">—</strong><small id="overview-links">—</small></article></div>
        <article class="panel"><div class="panel-head"><div><h3>运行时详情</h3><p>重连、拒绝和错误不会被静默吞掉。</p></div></div><pre id="overview-json">{}</pre></article>
      </section>
      <section id="nodes" class="page hidden"><div class="section-intro"><div><p class="eyebrow">AUTHORITY TREE</p><h2>节点与权限链</h2><p>父子关系同时定义链路信任边界；资源挂载在节点下。</p></div><button id="load-topology">读取拓扑</button></div><div id="topology-list" class="node-list empty">尚未读取拓扑</div></section>
      <section id="resources" class="page hidden"><div class="section-intro"><div><p class="eyebrow">RESOURCE CATALOG</p><h2>变量、流与指令</h2><p>从节点目录发现能力，不依赖旧子协议或服务 wrapper。</p></div><button id="load-catalog">读取目录</button></div><div class="split"><div id="catalog-list" class="resource-list empty">尚未读取资源</div><article class="panel console"><h3>资源控制台</h3><label>资源名<input id="resource-name" placeholder="system/health"></label><div class="row"><button id="snapshot">读取快照</button><button id="subscribe" class="secondary">订阅</button><button id="cancel-sub" class="ghost">取消订阅</button></div><label>Command JSON<textarea id="command-json" spellcheck="false">{"version":1}</textarea></label><button id="invoke">执行指令</button><pre id="resource-output">选择资源后开始操作</pre></article></div></section>
      <section id="metrics" class="page hidden"><div class="section-intro"><div><p class="eyebrow">METRICS NODE</p><h2>设备指标</h2><p>每项指标都是独立 Variable，可写指标通过对应 Command 控制。</p></div><button id="metrics-refresh">刷新</button></div><div class="grid two"><article class="panel"><h3>指标快照</h3><pre id="metrics-status">{}</pre></article><article class="panel"><h3>配置</h3><pre id="metrics-config">{}</pre></article></div><div class="row"><button class="resource-shortcut secondary" data-resource="metrics/config/update">配置指令</button></div></section>
      <section id="clipboard" class="page hidden"><div class="section-intro"><div><p class="eyebrow">CLIPBOARD NODE</p><h2>受控剪贴板同步</h2><p>正文只出现在受保护流或显式 Command 中，不写入状态和日志。</p></div><button id="clipboard-refresh">刷新</button></div><div class="grid two"><article class="panel"><h3>状态</h3><pre id="clipboard-status">{}</pre></article><article class="panel"><h3>配置</h3><pre id="clipboard-config">{}</pre></article></div><article class="panel"><label>发送正文<textarea id="clipboard-body" placeholder="正文不会进入桌面日志"></textarea></label><button id="clipboard-send">发送</button></article></section>
      <section id="files" class="page hidden"><div class="section-intro"><div><p class="eyebrow">FILE FEATURE</p><h2>文件传输</h2><p>有界分块、摘要校验、失败显式取消。</p></div><button id="files-refresh">刷新</button></div><article class="panel"><pre id="files-list">{}</pre></article><article class="panel form-grid"><label>本地源路径<input id="file-source" placeholder="D:\\data\\report.pdf"></label><label>目标相对路径<input id="file-destination" placeholder="inbox/report.pdf"></label><label>Content-Type<input id="file-type" value="application/octet-stream"></label><button id="file-upload">上传（最大 64 MiB）</button></article></section>
      <section id="flows" class="page hidden"><div class="section-intro"><div><p class="eyebrow">FLOW FEATURE</p><h2>自动化流</h2><p>定义与运行是 Variable，生命周期操作是 Command。</p></div><button id="flows-refresh">刷新</button></div><div class="grid two"><article class="panel"><h3>定义</h3><pre id="flow-definitions">{}</pre></article><article class="panel"><h3>运行</h3><pre id="flow-runs">{}</pre></article></div><article class="panel"><label>Flow Command<select id="flow-command"><option value="flow/create">创建</option><option value="flow/update">更新</option><option value="flow/run">运行</option><option value="flow/cancel">取消</option><option value="flow/archive">归档</option></select></label><label>请求 JSON<textarea id="flow-json" spellcheck="false">{"version":1}</textarea></label><button id="flow-invoke">执行</button><pre id="flow-output">等待操作</pre></article></section>
      <section id="authority" class="page hidden"><div class="section-intro"><div><p class="eyebrow">SERVER-SIDE DECISION</p><h2>权限与准入</h2><p>按钮可见性不是权限边界；最终结果始终来自父节点裁决。</p></div><button id="authority-refresh">读取管理配置</button></div><article class="panel"><pre id="authority-config">{}</pre></article><div class="grid two"><article class="panel form-grid"><h3>签发准入凭证</h3><label>子 NodeID<input id="permit-child"></label><label>子公钥<input id="permit-key"></label><label>角色<input id="permit-role" value="member"></label><label>有效期（毫秒）<input id="permit-ttl" value="3600000"></label><button id="permit-issue">签发</button><pre id="permit-output">—</pre></article><article class="panel form-grid"><h3>撤销节点</h3><label>NodeID<input id="revoke-node"></label><label>原因<input id="revoke-reason" value="operator requested"></label><button id="node-revoke" class="danger">撤销</button><pre id="revoke-output">—</pre></article></div></section>
      <section id="settings" class="page hidden"><div class="section-intro"><div><p class="eyebrow">VERSIONED STORAGE V1</p><h2>连接设置</h2><p>保存后重开本地身份会话；旧格式不会被静默误读。</p></div><button id="settings-save">保存并重开会话</button></div><article class="panel form-grid"><label>Profile<input id="setting-profile"></label><label>本机 NodeID<input id="setting-node"></label><label>父节点地址<input id="setting-endpoint" placeholder="127.0.0.1:9000"></label><label>父 NodeID<input id="setting-parent"></label><label class="wide">父节点 Ed25519 公钥<input id="setting-key"></label><label class="wide">准入凭证 JSON（首次加入时）<textarea id="setting-permit" spellcheck="false"></textarea></label></article></section>
      <section id="logs" class="page hidden"><div class="section-intro"><div><p class="eyebrow">OPERATION LOG</p><h2>桌面操作日志</h2><p>记录生命周期和错误，不记录剪贴板或文件正文。</p></div><button id="logs-refresh">刷新</button></div><div id="log-list" class="log-list empty">暂无日志</div></section>
    </main>
  </div>`

const element = <T extends HTMLElement>(selector: string): T => document.querySelector<T>(selector)!
const value = (selector: string): string => element<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(selector).value
const owner = (): number => {
  const parsed = Number(value('#owner-id'))
  if (!Number.isSafeInteger(parsed) || parsed <= 0) throw new Error('目标 NodeID 必须是正整数')
  return parsed
}
const output = (selector: string, data: unknown) => { element(selector).textContent = typeof data === 'string' ? data : pretty(data) }
const notice = (message: string, kind: 'ok' | 'error' = 'ok') => {
  const target = element('#notice')
  target.textContent = message
  target.className = kind
  window.setTimeout(() => { if (target.textContent === message) target.textContent = '' }, 5000)
}
const run = async <T>(label: string, action: () => Promise<T>): Promise<T | undefined> => {
  try {
    const result = await action()
    notice(`${label}完成`)
    return result
  } catch (error) {
    notice(`${label}失败：${errorText(error)}`, 'error')
    return undefined
  }
}

function showPage(page: Page) {
  currentPage = page
  document.querySelectorAll<HTMLElement>('.page').forEach((item) => item.classList.toggle('hidden', item.id !== page))
  document.querySelectorAll<HTMLButtonElement>('nav button').forEach((button) => button.classList.toggle('active', button.dataset.page === page))
  element('#page-title').textContent = pages.find((item) => item.id === page)?.label ?? page
}

async function refreshStatus() {
  const status = await api.status().catch((error) => ({ state: 'error', last_error: errorText(error) }))
  const state = String(status.state ?? 'unknown')
  element('#status-label').textContent = state === 'connected' ? '已连接' : state
  element('#status-detail').textContent = status.endpoint || status.last_error || '本地会话就绪'
  element('#status-dot').className = state
  element('#overview-state').textContent = state
  element('#overview-endpoint').textContent = status.endpoint || '未设置链路'
  return status
}

async function refreshOverview() {
  const status = await refreshStatus()
  const identity = await api.identity()
  element('#overview-node').textContent = `Node ${identity.node_id}`
  element('#overview-key').textContent = identity.public_key ? `${String(identity.public_key).slice(0, 18)}…` : '公钥不可用'
  let health: Record<string, unknown> = {}
  try { health = await api.health(owner()) } catch (error) { health = { state: 'unavailable', error: errorText(error) } }
  element('#overview-health').textContent = String(health.state ?? '不可用')
  element('#overview-links').textContent = health.active_links == null ? '连接后可读取' : `${health.active_links} 条活动链路`
  output('#overview-json', { status, identity, health })
}

async function loadTopology() {
  const topology = await api.topology(owner())
  const list = element('#topology-list')
  list.replaceChildren()
  const nodes = Array.isArray(topology.nodes) ? topology.nodes : []
  list.classList.toggle('empty', nodes.length === 0)
  if (nodes.length === 0) { list.textContent = '拓扑为空'; return }
  nodes.forEach((node: Record<string, unknown>) => {
    const card = document.createElement('article')
    const title = document.createElement('strong')
    title.textContent = String(node.display_name || `Node ${node.node_id}`)
    const meta = document.createElement('span')
    meta.textContent = `#${node.node_id} · ${node.role} · parent ${node.parent_id || 'root'}`
    card.append(title, meta)
    list.append(card)
  })
}

async function loadCatalog() {
  const catalog = await api.catalog(owner())
  const list = element('#catalog-list')
  list.replaceChildren()
  const resources = Array.isArray(catalog.resources) ? catalog.resources : []
  list.classList.toggle('empty', resources.length === 0)
  if (resources.length === 0) { list.textContent = '该节点没有公开资源'; return }
  resources.forEach((resource: Record<string, unknown>) => {
    const button = document.createElement('button')
    const title = document.createElement('strong')
    title.textContent = String(resource.name)
    const meta = document.createElement('span')
    meta.textContent = `${resource.kind} · ${resource.schema || 'schema unknown'}`
    button.append(title, meta)
    button.addEventListener('click', () => { element<HTMLInputElement>('#resource-name').value = String(resource.name); output('#resource-output', resource) })
    list.append(button)
  })
}

async function startSubscription(name = value('#resource-name')) {
  if (activeSubscription) await api.cancel(activeSubscription)
  activeSubscription = await api.subscribe(owner(), name, 60_000)
  polling = true
  output('#resource-output', { kind: 'ready', subscription_id: activeSubscription, resource: name })
  while (polling && activeSubscription) {
    const result = await api.poll(activeSubscription, 15_000).catch((error) => ({ kind: 'error', error: errorText(error) }))
    if (!polling) break
    if (result.kind !== 'timeout') output('#resource-output', result)
    if (result.kind === 'closed' || result.kind === 'error') break
  }
}

async function cancelSubscription() {
  polling = false
  if (activeSubscription) await api.cancel(activeSubscription)
  activeSubscription = 0
  output('#resource-output', '订阅已取消')
}

async function refreshMetrics() {
  const catalog = await api.catalog(owner())
  const variables = (Array.isArray(catalog.resources) ? catalog.resources : []).filter((item: Record<string, unknown>) => item.kind === 'variable' && String(item.name).startsWith('metrics/') && item.name !== 'metrics/config')
  const samples = await Promise.all(variables.map(async (item: Record<string, unknown>) => [String(item.name), await api.snapshot(owner(), String(item.name))]))
  const config = await api.snapshot(owner(), 'metrics/config')
  output('#metrics-status', Object.fromEntries(samples)); output('#metrics-config', config)
}

async function refreshClipboard() {
  const [status, config] = await Promise.all([api.snapshot(owner(), 'clipboard/status'), api.snapshot(owner(), 'clipboard/config')])
  output('#clipboard-status', status); output('#clipboard-config', config)
}

async function loadSettings() {
  settings = await api.settings()
  element<HTMLInputElement>('#setting-profile').value = settings.profile
  element<HTMLInputElement>('#setting-node').value = settings.node_id
  element<HTMLInputElement>('#setting-endpoint').value = settings.endpoint || ''
  element<HTMLInputElement>('#setting-parent').value = settings.parent_node_id || ''
  element<HTMLInputElement>('#setting-key').value = settings.parent_public_key || ''
  element<HTMLTextAreaElement>('#setting-permit').value = settings.permit_json || ''
}

async function saveSettings() {
  const next: Settings = {
    ...settings,
    version: 1,
    profile: value('#setting-profile'),
    node_id: value('#setting-node'),
    endpoint: value('#setting-endpoint'),
    parent_node_id: value('#setting-parent'),
    parent_public_key: value('#setting-key'),
    permit_json: value('#setting-permit'),
  }
  settings = await api.saveSettings(next)
  await refreshOverview()
}

async function refreshLogs() {
  const logs = await api.logs()
  const list = element('#log-list')
  list.replaceChildren()
  if (!Array.isArray(logs) || logs.length === 0) { list.textContent = '暂无日志'; list.classList.add('empty'); return }
  list.classList.remove('empty')
  logs.slice().reverse().forEach((entry: Record<string, unknown>) => {
    const row = document.createElement('article')
    const time = document.createElement('time')
    time.textContent = new Date(Number(entry.time_unix_ms)).toLocaleTimeString()
    const level = document.createElement('strong')
    level.textContent = String(entry.level)
    const message = document.createElement('span')
    message.textContent = String(entry.message)
    row.append(time, level, message); list.append(row)
  })
}

document.querySelectorAll<HTMLButtonElement>('nav button').forEach((button) => button.addEventListener('click', () => showPage(button.dataset.page as Page)))
element('#refresh').addEventListener('click', () => run('刷新', async () => {
  if (currentPage === 'overview') return refreshOverview()
  if (currentPage === 'nodes') return loadTopology()
  if (currentPage === 'resources') return loadCatalog()
  if (currentPage === 'metrics') return refreshMetrics()
  if (currentPage === 'clipboard') return refreshClipboard()
  if (currentPage === 'files') return api.fileTransfers(owner()).then((data) => output('#files-list', data))
  if (currentPage === 'flows') return Promise.all([api.flowDefinitions(owner()).then((data) => output('#flow-definitions', data)), api.flowRuns(owner()).then((data) => output('#flow-runs', data))])
  if (currentPage === 'logs') return refreshLogs()
}))
element('#connect').addEventListener('click', () => run('连接', async () => { await api.connect(); await refreshStatus() }))
element('#disconnect').addEventListener('click', () => run('断开', async () => { await api.disconnect(); await refreshStatus() }))
element('#load-topology').addEventListener('click', () => run('读取拓扑', loadTopology))
element('#load-catalog').addEventListener('click', () => run('读取目录', loadCatalog))
element('#snapshot').addEventListener('click', () => run('读取快照', async () => output('#resource-output', await api.snapshot(owner(), value('#resource-name')))))
element('#invoke').addEventListener('click', () => run('执行指令', async () => output('#resource-output', await api.invoke(owner(), value('#resource-name'), JSON.parse(value('#command-json'))))))
element('#subscribe').addEventListener('click', () => run('订阅', () => startSubscription()))
element('#cancel-sub').addEventListener('click', () => run('取消订阅', cancelSubscription))
element('#metrics-refresh').addEventListener('click', () => run('刷新指标', refreshMetrics))
element('#clipboard-refresh').addEventListener('click', () => run('刷新剪贴板状态', refreshClipboard))
element('#clipboard-send').addEventListener('click', () => run('发送剪贴板', async () => api.invoke(owner(), 'clipboard/send', { version: 1, text: value('#clipboard-body') })))
document.querySelectorAll<HTMLButtonElement>('.resource-shortcut').forEach((button) => button.addEventListener('click', () => { showPage('resources'); element<HTMLInputElement>('#resource-name').value = button.dataset.resource || '' }))
element('#files-refresh').addEventListener('click', () => run('刷新文件传输', async () => output('#files-list', await api.fileTransfers(owner()))))
element('#file-upload').addEventListener('click', () => run('上传文件', async () => output('#files-list', await api.uploadFile(owner(), value('#file-source'), value('#file-destination'), value('#file-type')))))
element('#flows-refresh').addEventListener('click', () => run('刷新自动化流', async () => { output('#flow-definitions', await api.flowDefinitions(owner())); output('#flow-runs', await api.flowRuns(owner())) }))
element('#flow-invoke').addEventListener('click', () => run('执行 Flow 指令', async () => output('#flow-output', await api.invoke(owner(), value('#flow-command'), JSON.parse(value('#flow-json'))))))
element('#authority-refresh').addEventListener('click', () => run('读取管理配置', async () => output('#authority-config', await api.managementConfig(owner()))))
element('#permit-issue').addEventListener('click', () => run('签发凭证', async () => output('#permit-output', await api.issuePermit(owner(), { version: 1, child_node_id: value('#permit-child'), child_public_key: value('#permit-key'), role: value('#permit-role'), ttl_ms: Number(value('#permit-ttl')) }))))
element('#node-revoke').addEventListener('click', () => run('撤销节点', async () => output('#revoke-output', await api.revokeNode(owner(), { version: 1, node_id: value('#revoke-node'), reason: value('#revoke-reason') }))))
element('#settings-save').addEventListener('click', () => run('保存设置', saveSettings))
element('#logs-refresh').addEventListener('click', () => run('刷新日志', refreshLogs))

showPage('overview')
void run('初始化', async () => { await loadSettings(); await refreshOverview() })
window.setInterval(() => { void refreshStatus() }, 3000)
