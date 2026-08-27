import './style.css'
import * as api from './api'
import { Configuration, Sample, Status } from './contracts'
import { metricLabel, normalizeChannels, sampleValue } from './model'

const root = document.querySelector<HTMLElement>('#app')!
let current: Status = {version: 1, running: false, samples: [], notifications: {queue_depth: 0, dropped: 0}}
let timer: number | undefined

root.innerHTML = `
  <header><div><p class="eyebrow">MYFLOWHUB / NODE</p><h1>Metrics</h1></div><span id="state" class="pill">已停止</span></header>
  <section class="connect panel">
    <label>状态目录<input id="stateDirectory" value="./state/metrics-windows" /></label>
    <label>节点 ID<input id="nodeID" value="20" inputmode="numeric" /></label>
    <label>父节点 ID<input id="parentID" value="1" inputmode="numeric" /></label>
    <label>TCP 端点<input id="endpoint" value="127.0.0.1:7341" /></label>
    <label class="wide">父节点公钥<input id="parentKey" placeholder="raw-base64 Ed25519 public key" /></label>
    <label class="wide">一次性接入凭证（可选）<textarea id="permit" rows="3" placeholder="JSON provisioning permit"></textarea></label>
    <div class="actions"><button id="identity">生成 / 读取身份</button><button id="start" class="primary">连接</button><button id="stop">停止</button></div>
    <pre id="identityOutput" class="wide output"></pre><p id="error" class="wide error"></p>
  </section>
  <section><div class="section-title"><div><p class="eyebrow">LIVE VARIABLES</p><h2>资源状态</h2></div><span id="connection"></span></div><div id="samples" class="metrics"></div></section>
  <section class="panel"><div class="section-title"><div><p class="eyebrow">POLICY</p><h2>采集、控制与通知</h2></div><button id="save">保存配置</button></div><div class="channel"><label>通知频道（逗号或换行分隔）<input id="notificationChannels" /></label></div><div id="settings" class="settings"></div></section>
`

const field = (id: string) => document.querySelector<HTMLInputElement>(`#${id}`)!
const error = document.querySelector<HTMLElement>('#error')!

document.querySelector('#identity')!.addEventListener('click', () => run(async () => {
  const result = await api.identity(field('stateDirectory').value, field('nodeID').value)
  document.querySelector<HTMLElement>('#identityOutput')!.textContent = `节点 ${result.node_id}\n公钥 ${result.public_key}`
}))

document.querySelector('#start')!.addEventListener('click', () => run(async () => {
  const permitText = document.querySelector<HTMLTextAreaElement>('#permit')!.value.trim()
  current = await api.start({
    state_directory: field('stateDirectory').value,
    node_id: field('nodeID').value,
    parent_node_id: field('parentID').value,
    endpoint: field('endpoint').value,
    parent_public_key: field('parentKey').value.trim(),
    ...(permitText ? {permit: JSON.parse(permitText)} : {}),
  })
  render()
  schedule()
}))

document.querySelector('#stop')!.addEventListener('click', () => run(async () => {
  await api.stop()
  current = await api.status()
  render()
}))

document.querySelector('#save')!.addEventListener('click', () => run(async () => {
  if (!current.configuration) throw new Error('节点尚未运行')
  const settings = [...document.querySelectorAll<HTMLElement>('[data-metric]')].map(row => ({
    metric: row.dataset.metric!,
    enabled: row.querySelector<HTMLInputElement>('[data-field="enabled"]')!.checked,
    writable: row.querySelector<HTMLInputElement>('[data-field="writable"]')!.checked,
    interval_ms: Number(row.querySelector<HTMLInputElement>('[data-field="interval"]')!.value),
  }))
  const notification_channels = normalizeChannels(field('notificationChannels').value)
  current.configuration = await api.updateConfiguration({...current.configuration, settings, notification_channels})
  renderSettings(current.configuration)
}))

async function refresh(): Promise<void> {
  current = await api.status()
  render()
}

function schedule(): void {
  if (timer !== undefined) window.clearInterval(timer)
  timer = window.setInterval(() => run(refresh), 1000)
}

async function run(action: () => Promise<void>): Promise<void> {
  error.textContent = ''
  try { await action() } catch (reason) { error.textContent = reason instanceof Error ? reason.message : String(reason) }
}

function render(): void {
  const state = document.querySelector<HTMLElement>('#state')!
  state.textContent = current.connection?.state ?? (current.running ? '启动中' : '已停止')
  state.dataset.state = current.connection?.state ?? 'stopped'
  document.querySelector<HTMLElement>('#connection')!.textContent = current.connection
    ? `${current.connection.endpoint} · 第 ${current.connection.attempt} 次尝试${current.connection.last_error ? ` · ${current.connection.last_error}` : ''}`
    : '未连接'
  const samples = document.querySelector<HTMLElement>('#samples')!
  samples.innerHTML = current.samples.map(renderSample).join('') || '<p class="empty">启动节点后显示资源变量</p>'
  if (current.configuration) renderSettings(current.configuration)
  else document.querySelector<HTMLElement>('#settings')!.innerHTML = '<p class="empty">节点尚未运行</p>'
}

function renderSample(sample: Sample): string {
  const value = escapeHTML(sampleValue(sample))
  return `<article class="metric"><div><span class="dot ${sample.status}"></span>${metricLabel(sample.metric)}</div><strong>${value}</strong><small>${sample.unit} · ${sample.status}${sample.error ? ` · ${escapeHTML(sample.error)}` : ''}</small></article>`
}

function renderSettings(configuration: Configuration): void {
  const channels = field('notificationChannels')
  if (document.activeElement !== channels) channels.value = configuration.notification_channels.join(', ')
  document.querySelector<HTMLElement>('#settings')!.innerHTML = configuration.settings.map(setting => `
    <div class="setting" data-metric="${setting.metric}"><span>${metricLabel(setting.metric)}</span>
      <label class="check"><input data-field="enabled" type="checkbox" ${setting.enabled ? 'checked' : ''}/>采集</label>
      <label class="check"><input data-field="writable" type="checkbox" ${setting.writable ? 'checked' : ''}/>可控制</label>
      <label class="interval"><input data-field="interval" type="number" min="250" max="86400000" value="${setting.interval_ms}"/> ms</label>
    </div>`).join('')
}

function escapeHTML(value: string): string { const node = document.createElement('span'); node.textContent = value; return node.innerHTML }
render()
