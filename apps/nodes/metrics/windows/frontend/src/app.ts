import type { MetricsAPI } from './api'
import type { StartRequest, Status } from './contracts'
import {
  parseNodeID,
  parsePermit,
  reconcileSelectedMetric,
  requireText,
  type Theme,
} from './model'
import { createMetricsView, type BusyAction, type MetricsViewState, type PageName } from './view'

const THEME_KEY = 'myflowhub.metrics.theme.v1'
const DEFAULT_POLL_INTERVAL_MS = 1000

interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

export interface MetricsAppEnvironment {
  storage?: StorageLike
  now?: () => number
  setTimeout?: (handler: TimerHandler, timeout?: number) => number
  clearTimeout?: (id: number) => void
  pollIntervalMS?: number
}

export interface MetricsApp {
  ready: Promise<void>
  dispose(): void
}

function emptyStatus(): Status {
  return {version: 1, running: false, samples: [], notifications: {queue_depth: 0, dropped: 0}}
}

function defaultStorage(): StorageLike | undefined {
  try {
    return window.localStorage
  } catch {
    return undefined
  }
}

function storedTheme(storage: StorageLike | undefined): Theme {
  if (!storage) return 'light'
  try {
    return storage.getItem(THEME_KEY) === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

function errorMessage(reason: unknown): string {
  if (reason instanceof Error && reason.message.trim()) return reason.message
  const value = String(reason).trim()
  return value || '发生未知错误'
}

export function createMetricsApp(root: HTMLElement, api: MetricsAPI, environment: MetricsAppEnvironment = {}): MetricsApp {
  const storage = environment.storage ?? defaultStorage()
  const now = environment.now ?? Date.now
  const scheduleTimeout = environment.setTimeout ?? window.setTimeout.bind(window)
  const cancelTimeout = environment.clearTimeout ?? window.clearTimeout.bind(window)
  const pollIntervalMS = environment.pollIntervalMS ?? DEFAULT_POLL_INTERVAL_MS
  if (!Number.isFinite(pollIntervalMS) || pollIntervalMS <= 0) throw new Error('pollIntervalMS 必须是正数')

  let state: MetricsViewState = {
    activePage: 'resources',
    theme: storedTheme(storage),
    status: emptyStatus(),
    definitions: [],
    nodeID: '20',
    busy: 'initializing',
  }
  let disposed = false
  let pollTimer: number | undefined
  let noticeTimer: number | undefined
  let statusEpoch = 0
  let refreshInFlight = false

  const view = createMetricsView(root, {
    navigate,
    refresh: () => { void refreshStatus(true) },
    toggleTheme,
    selectMetric,
    savePolicy: () => { void savePolicy() },
    readIdentity: () => { void readIdentity() },
    start: () => { void start() },
    stop: () => { void stop() },
    dismissError,
  })

  function render(): void {
    if (!disposed) view.render(state)
  }

  function navigate(page: PageName): void {
    if (disposed) return
    state = {...state, activePage: page}
    render()
  }

  function toggleTheme(): void {
    if (disposed) return
    const theme: Theme = state.theme === 'light' ? 'dark' : 'light'
    state = {...state, theme, error: undefined}
    try {
      storage?.setItem(THEME_KEY, theme)
    } catch {
      state = {...state, error: '主题已切换，但无法保存主题偏好'}
    }
    render()
  }

  function selectMetric(metric: string): void {
    if (disposed || !state.status.samples.some(sample => sample.metric === metric)) return
    state = {...state, selectedMetric: metric}
    render()
  }

  function dismissError(): void {
    if (disposed) return
    state = {...state, error: undefined}
    render()
  }

  function applyStatus(status: Status): void {
    state = {
      ...state,
      status,
      selectedMetric: reconcileSelectedMetric(status.samples, state.selectedMetric),
      lastRefreshUnixMS: now(),
    }
  }

  function cancelPolling(): void {
    if (pollTimer !== undefined) cancelTimeout(pollTimer)
    pollTimer = undefined
  }

  function syncPolling(): void {
    if (disposed || !state.status.running) {
      cancelPolling()
      return
    }
    if (pollTimer !== undefined) return
    pollTimer = scheduleTimeout(() => {
      pollTimer = undefined
      void refreshStatus(false).finally(syncPolling)
    }, pollIntervalMS)
  }

  function clearNotice(): void {
    if (noticeTimer !== undefined) cancelTimeout(noticeTimer)
    noticeTimer = undefined
    state = {...state, notice: undefined}
  }

  function queueNotice(message: string): void {
    clearNotice()
    state = {...state, notice: message}
    noticeTimer = scheduleTimeout(() => {
      noticeTimer = undefined
      if (disposed) return
      state = {...state, notice: undefined}
      render()
    }, 3600)
  }

  async function initialize(): Promise<void> {
    render()
    const [statusResult, definitionsResult] = await Promise.allSettled([api.status(), api.definitions()])
    if (disposed) return
    const errors: string[] = []
    if (statusResult.status === 'fulfilled') applyStatus(statusResult.value)
    else errors.push(`读取状态失败：${errorMessage(statusResult.reason)}`)
    if (definitionsResult.status === 'fulfilled') state = {...state, definitions: definitionsResult.value}
    else errors.push(`读取指标定义失败：${errorMessage(definitionsResult.reason)}`)
    state = {...state, busy: undefined, error: errors.length ? errors.join('；') : undefined}
    render()
    syncPolling()
  }

  async function refreshStatus(userInitiated: boolean): Promise<void> {
    if (disposed || refreshInFlight || state.busy) return
    refreshInFlight = true
    const epoch = statusEpoch
    if (userInitiated) {
      clearNotice()
      state = {...state, busy: 'refresh', error: undefined}
      render()
    }
    try {
      const status = await api.status()
      if (disposed || epoch !== statusEpoch) return
      applyStatus(status)
      if (userInitiated) queueNotice('状态已刷新')
    } catch (reason) {
      if (!disposed && epoch === statusEpoch) state = {...state, error: `刷新状态失败：${errorMessage(reason)}`}
    } finally {
      refreshInFlight = false
      if (userInitiated && !disposed && state.busy === 'refresh') state = {...state, busy: undefined}
      render()
      syncPolling()
    }
  }

  async function runAction(action: Exclude<BusyAction, 'initializing' | 'refresh' | undefined>, work: () => Promise<void>, successMessage: string): Promise<void> {
    if (disposed || state.busy) return
    statusEpoch += 1
    clearNotice()
    state = {...state, busy: action, error: undefined}
    render()
    try {
      await work()
      if (!disposed) queueNotice(successMessage)
    } catch (reason) {
      if (!disposed) state = {...state, error: errorMessage(reason)}
    } finally {
      if (!disposed) {
        state = {...state, busy: undefined}
        render()
        syncPolling()
      }
    }
  }

  async function readIdentity(): Promise<void> {
    await runAction('identity', async () => {
      const fields = view.readIdentityFields()
      const stateDirectory = requireText(fields.stateDirectory, '状态目录')
      const nodeID = parseNodeID(fields.nodeID, '节点 ID')
      const identity = await api.identity(stateDirectory, nodeID)
      state = {...state, identity, nodeID: identity.node_id}
    }, '身份已读取')
  }

  async function start(): Promise<void> {
    await runAction('start', async () => {
      const fields = view.readConnectionFields()
      const permit = parsePermit(fields.permit)
      const request: StartRequest = {
        state_directory: requireText(fields.stateDirectory, '状态目录'),
        node_id: parseNodeID(fields.nodeID, '节点 ID'),
        parent_node_id: parseNodeID(fields.parentNodeID, '父节点 ID'),
        endpoint: requireText(fields.endpoint, 'TCP 端点'),
        parent_public_key: requireText(fields.parentPublicKey, '父节点公钥'),
        ...(permit ? {permit} : {}),
      }
      const status = await api.start(request)
      applyStatus(status)
      state = {...state, nodeID: request.node_id}
      view.clearPermit()
    }, '节点连接已启动')
  }

  async function stop(): Promise<void> {
    await runAction('stop', async () => {
      cancelPolling()
      await api.stop()
      applyStatus(await api.status())
    }, '节点已停止')
  }

  async function savePolicy(): Promise<void> {
    await runAction('save', async () => {
      const current = state.status.configuration
      if (!current) throw new Error('节点尚未运行，无法保存采集策略')
      const updated = await api.updateConfiguration(view.readConfiguration(current))
      state = {...state, status: {...state.status, configuration: updated}}
    }, '采集策略已保存')
  }

  const ready = initialize()
  return {
    ready,
    dispose: () => {
      if (disposed) return
      disposed = true
      statusEpoch += 1
      cancelPolling()
      if (noticeTimer !== undefined) cancelTimeout(noticeTimer)
      noticeTimer = undefined
      view.clearPermit()
    },
  }
}
