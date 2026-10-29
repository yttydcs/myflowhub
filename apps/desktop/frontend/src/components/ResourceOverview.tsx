import { memo, useContext, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react'
import { List, Radio, Search } from 'lucide-react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor } from '../types'
import type { VariableController } from '../variable/controller'
import { VariableProfileContext } from '../variable/useVariable'
import { VariableOverviewStore } from '../variable/overview-store'
import { resourceKey, errorText } from '../lib/utils'
import { Button } from './ui/button'
import './resource-overview.css'

const ROW_HEIGHT = 44
const noopSubscribe = () => () => {}
const noSnapshot = () => undefined
function preview(value: unknown, depth = 0): string {
  if (value === undefined) return '—'
  if (typeof value === 'string') return value.length > 96 ? `${value.slice(0, 96)}…` : value
  if (value === null || typeof value !== 'object') return String(value)
  if (depth >= 2) return Array.isArray(value) ? '[…]' : '{…}'
  if (Array.isArray(value)) return `[${value.slice(0, 4).map((item) => preview(item, depth + 1)).join(', ')}${value.length > 4 ? ', …' : ''}]`
  const keys = Object.keys(value)
  return `{${keys.slice(0, 4).map((key) => `${key}: ${preview((value as Record<string, unknown>)[key], depth + 1)}`).join(', ')}${keys.length > 4 ? ', …' : ''}}`
}

const OverviewRow = memo(function OverviewRow({ resource, controller, index, onSelect, onWatch, onFocus }: {
  resource: ResourceDescriptor; controller?: VariableController; index: number
  onSelect(resource: ResourceDescriptor): void; onWatch(resource: ResourceDescriptor): void; onFocus(index: number): void
}) {
  const state = useSyncExternalStore(controller?.subscribe || noopSubscribe, controller?.getSnapshot || noSnapshot)
  const text = useMemo(() => preview(state?.snapshot?.value), [state?.snapshot?.value])
  // A failed write can block editing while the authorized subscription stays live.
  const current = state?.live && state.snapshot?.status === 'ready'
  const status = current ? (state.blocked || state.error ? '实时 · 操作异常' : '实时') : state?.blocked || state?.error ? '不可用' : state?.snapshot?.status === 'stale' ? '已过期' : controller ? '等待快照' : '未订阅'
  return <div className="overview-row" role="row" aria-rowindex={index + 2} style={{ top: index * ROW_HEIGHT }} onFocus={() => onFocus(index)}>
    <div role="cell"><button type="button" className="overview-watch" aria-label={`${controller ? '停止订阅' : '订阅'} ${resource.id.name}`} aria-pressed={Boolean(controller)} onClick={() => onWatch(resource)}><Radio size={14} aria-hidden="true" /></button></div>
    <div role="cell"><button type="button" className="overview-name" onClick={() => onSelect(resource)} title={`${resource.id.owner_node_id}/${resource.id.name}`}>{resource.presentation?.label || resource.id.name}<small>{resource.id.name}</small></button></div>
    <div role="cell" className="overview-owner">{resource.id.owner_node_id}</div>
    <div role="cell" className="overview-type">{state?.snapshot?.definition?.type || 'Variable'}</div>
    <div role="cell" className="overview-value" title={text}>{text}</div>
    <div role="cell"><span className={`overview-status ${current ? 'is-live' : ''}`} title={state?.blocked || state?.error || state?.observation}>{status}</span></div>
  </div>
})

export function ResourceOverview({ api, resources, hidden, onSelect }: { api: DesktopAPI; resources: ResourceDescriptor[]; hidden: boolean; onSelect(resource: ResourceDescriptor): void }) {
  const profile = useContext(VariableProfileContext)
  const store = useMemo(() => new VariableOverviewStore(api, profile), [api, profile])
  const summary = useSyncExternalStore(store.subscribe, store.getSnapshot)
  const [query, setQuery] = useState('')
  const [error, setError] = useState('')
  const [scrollTop, setScrollTop] = useState(0)
  const [height, setHeight] = useState(480)
  const [focused, setFocused] = useState<number>()
  const viewport = useRef<HTMLDivElement>(null)
  useEffect(() => () => store.stop(), [store])
  useEffect(() => {
    const node = viewport.current
    if (!node || hidden) return
    const measure = () => { if (node.clientHeight) setHeight(node.clientHeight) }
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(node)
    return () => observer.disconnect()
  }, [hidden])
  const variables = useMemo(() => {
    const known = new Map(summary.resources.map((resource) => [resourceKey(resource.id.owner_node_id, resource.id.name), resource]))
    for (const resource of resources) known.set(resourceKey(resource.id.owner_node_id, resource.id.name), resource)
    return [...known.values()].filter((resource) => resource.type === 'mfh.variable' && resource.type_version === 2)
  }, [resources, summary.resources])
  const filtered = useMemo(() => {
    const text = query.trim().toLocaleLowerCase()
    return variables.filter((resource) => !text || `${resource.id.owner_node_id} ${resource.id.name} ${resource.presentation?.label || ''}`.toLocaleLowerCase().includes(text))
  }, [variables, query])
  const start = Math.max(0, Math.min(filtered.length, Math.floor(scrollTop / ROW_HEIGHT) - 8))
  const end = Math.min(filtered.length, start + Math.ceil(height / ROW_HEIGHT) + 16)
  const indices = Array.from({ length: end - start }, (_, index) => start + index)
  if (focused !== undefined && focused < filtered.length && !indices.includes(focused)) indices.push(focused)
  indices.sort((a, b) => a - b)
  const watch = useMemo(() => (resource: ResourceDescriptor) => {
    try {
      const controller = store.controller(resource)
      if (controller) {
        if (controller.getSnapshot().dirty && !window.confirm('停止订阅可能丢弃此资源的未提交草稿，是否继续？')) return
        store.unwatch(resource)
      } else store.watch([resource])
      setError('')
    } catch (current) { setError(errorText(current)) }
  }, [store])
  return <section className="resource-overview" hidden={hidden} aria-label="资源总览">
    <header className="overview-header"><div><h2><List size={17} aria-hidden="true" />资源总览</h2></div><div className="overview-count"><strong>{summary.live.toLocaleString()}</strong><span>/ {summary.count.toLocaleString()} 实时</span>{summary.failed > 0 && <span className="overview-failures">{summary.failed} 项异常</span>}</div></header>
    <div className="overview-toolbar"><label><Search size={14} aria-hidden="true" /><input aria-label="筛选总览变量" value={query} placeholder="筛选资源名、标签或所有者" onChange={(event) => { setQuery(event.target.value); setScrollTop(0); setFocused(undefined); if (viewport.current) viewport.current.scrollTop = 0 }} /></label><Button size="sm" onClick={() => { try { store.watch(filtered); setError('') } catch (current) { setError(errorText(current)) } }}>订阅筛选结果</Button><Button size="sm" variant="ghost" disabled={!summary.count} onClick={() => { if (!store.hasDrafts || window.confirm('停止全部订阅可能丢弃未提交草稿，是否继续？')) store.stop() }}>停止全部</Button></div>
    {error && <p className="overview-error" role="alert">{error}</p>}
    <div role="table" aria-label="标准变量列表" aria-rowcount={filtered.length + 1} className="overview-table">
      <div role="row" className="overview-columns"><span role="columnheader" aria-label="订阅" /><span role="columnheader">资源</span><span role="columnheader">所有者</span><span role="columnheader">类型</span><span role="columnheader">当前值</span><span role="columnheader">状态</span></div>
      <div ref={viewport} className="overview-viewport" role="rowgroup" tabIndex={0} aria-label="滚动浏览变量" onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}><div className="overview-spacer" style={{ height: filtered.length * ROW_HEIGHT }}>{indices.map((index) => {
        const resource = filtered[index]!
        return <OverviewRow key={resourceKey(resource.id.owner_node_id, resource.id.name)} resource={resource} controller={store.controller(resource)} index={index} onSelect={onSelect} onWatch={watch} onFocus={setFocused} />
      })}</div>{!filtered.length && <p className="overview-empty">当前没有匹配变量。请在资源浏览器加载节点目录，或调整筛选条件。</p>}</div>
    </div>
  </section>
}
