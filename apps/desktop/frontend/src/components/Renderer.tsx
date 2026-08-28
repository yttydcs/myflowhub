import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Braces, FileUp, LoaderCircle, Play, Radio, RefreshCw, Send, SquareTerminal } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { errorText } from '../lib/utils'
import type { ResourceDescriptor, ResourceEvent, TopologyNode } from '../types'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Input, Textarea } from './ui/input'

function pretty(value: unknown) { return JSON.stringify(value, null, 2) }

function encodeJSONBytes(value: unknown) {
  const bytes = new TextEncoder().encode(JSON.stringify(value))
  let binary = ''
  bytes.forEach((byte) => { binary += String.fromCharCode(byte) })
  return btoa(binary)
}

function decodeEventPayload(event: ResourceEvent): unknown {
  if (!event.value) return null
  try {
    const binary = atob(event.value)
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0))
    return JSON.parse(new TextDecoder().decode(bytes))
  } catch {
    return event.value
  }
}

function StateMessage({ kind, children }: { kind?: 'error' | 'loading'; children: React.ReactNode }) {
  return <div className={`renderer-state ${kind || ''}`}>{kind === 'error' ? <AlertTriangle size={17} /> : <LoaderCircle size={17} />}{children}</div>
}

function VariableRenderer({ api, resource }: RendererProps) {
  const [value, setValue] = useState<unknown>()
  const [revision, setRevision] = useState(0)
  const [input, setInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [writing, setWriting] = useState(false)
  const writable = resource.capabilities.find((item) => item.name === 'write')
  const refresh = async () => {
    setLoading(true); setError('')
    try {
      if (writable) {
        const subscriptionID = await api.subscribe(resource.id.owner_node_id, resource.id.name, 'subscribe', 5_000)
        try {
          const result = await api.poll(subscriptionID, 5_000)
          if (result.kind !== 'event') throw new Error('未收到 Variable 快照')
          const event = result.payload as ResourceEvent
          const next = decodeEventPayload(event)
          setRevision(event.revision || 0); setValue(next); setInput(pretty(next))
        } finally { await api.cancel(subscriptionID) }
      } else {
        const next = await api.snapshot(resource.id.owner_node_id, resource.id.name)
        setValue(next); setInput(pretty(next))
      }
    }
    catch (current) { setError(errorText(current)) }
    finally { setLoading(false) }
  }
  const write = async () => {
    setWriting(true); setError('')
    try {
      const next = JSON.parse(input)
      await api.operate(resource.id.owner_node_id, resource.id.name, 'write', writable?.input_schema || 'mfh.variable-write.v2', {
        version: 2, expected_revision: revision, value: encodeJSONBytes(next),
      })
      await refresh()
    } catch (current) { setError(errorText(current)) }
    finally { setWriting(false) }
  }
  useEffect(() => { void refresh() }, [resource.id.owner_node_id, resource.id.name])
  if (loading) return <StateMessage kind="loading">正在读取快照…</StateMessage>
  if (error) return <StateMessage kind="error">{error}</StateMessage>
  return <div className="value-renderer">
    {writable ? <label>当前值 · revision {revision}<Textarea value={input} onChange={(event) => setInput(event.target.value)} rows={5} spellCheck={false} /></label> : <pre>{pretty(value)}</pre>}
    <div className="value-actions"><Button variant="ghost" size="sm" onClick={() => void refresh()}><RefreshCw size={14} />刷新</Button>{writable && <Button size="sm" disabled={writing || revision === 0} onClick={() => void write()}>{writing ? '写入中…' : '条件写入'}</Button>}</div>
  </div>
}

function EventRenderer({ api, resource }: RendererProps) {
  const [events, setEvents] = useState<ResourceEvent[]>([])
  const [state, setState] = useState<'connecting' | 'live' | 'stopped'>('connecting')
  const [error, setError] = useState('')
  useEffect(() => {
    let active = true
    let subscriptionID = 0
    void (async () => {
      try {
        subscriptionID = await api.subscribe(resource.id.owner_node_id, resource.id.name, 'subscribe')
        if (!active) { await api.cancel(subscriptionID); return }
        setState('live')
        while (active) {
          const result = await api.poll(subscriptionID, 1000)
          if (!active) break
          if (result.kind === 'event') setEvents((current) => [result.payload as ResourceEvent, ...current].slice(0, 20))
          if (result.kind === 'error') throw new Error((result.payload as { message?: string })?.message || '订阅失败')
          if (result.kind === 'closed') break
        }
      } catch (current) {
        if (active) setError(errorText(current))
      } finally {
        if (active) setState('stopped')
      }
    })()
    return () => { active = false; if (subscriptionID) void api.cancel(subscriptionID) }
  }, [api, resource.id.owner_node_id, resource.id.name])
  if (error) return <StateMessage kind="error">{error}</StateMessage>
  return <div className="event-renderer">
    <div className="live-line"><span className={`live-dot ${state}`} />{state === 'live' ? '实时订阅中' : '正在建立订阅'}<Badge>{events.length} events</Badge></div>
    <div className="event-list">{events.length === 0 ? <p className="empty-copy">等待第一条事件</p> : events.map((event, index) => (
      <article key={`${event.sequence || 0}-${index}`}><small>#{event.sequence || event.revision || '—'}{event.publisher_node_id ? ` · publisher ${event.publisher_node_id}/${event.publisher_sequence || 0}` : ''} · {event.schema || 'raw'}</small><pre>{pretty(decodeEventPayload(event))}</pre></article>
    ))}</div>
  </div>
}

function OperationRenderer({ api, resource, capability = 'invoke' }: RendererProps & { capability?: string }) {
  const descriptor = resource.capabilities.find((item) => item.name === capability)
  const [input, setInput] = useState('{}')
  const [result, setResult] = useState<unknown>()
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    setBusy(true); setError('')
    try {
      const value = JSON.parse(input)
      setResult(await api.operate(resource.id.owner_node_id, resource.id.name, capability, descriptor?.input_schema || '', value))
    } catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }
  return <div className="operation-renderer">
    <label>输入 · {descriptor?.input_schema || 'application/json'}<Textarea value={input} onChange={(event) => setInput(event.target.value)} rows={5} spellCheck={false} /></label>
    {error && <p className="form-error" role="alert">{error}</p>}
    <Button size="sm" disabled={busy} onClick={() => void submit()}><Play size={14} />{busy ? '执行中…' : `执行 ${capability}`}</Button>
    {result !== undefined && <pre className="result-block">{pretty(result)}</pre>}
  </div>
}

function FileRenderer({ api, resource }: RendererProps) {
  const [source, setSource] = useState('')
  const [destination, setDestination] = useState('')
  const [result, setResult] = useState<unknown>()
  const [error, setError] = useState('')
  const upload = async () => {
    setError('')
    try { setResult(await api.uploadFile(resource.id.owner_node_id, source, destination, 'application/octet-stream')) }
    catch (current) { setError(errorText(current)) }
  }
  return <div className="file-renderer">
    <FileUp size={26} />
    <p>文件通过绑定到此资源的有界 session 传输；控制帧不会被数据块淹没。</p>
    <label>本地文件路径<Input value={source} onChange={(event) => setSource(event.target.value)} /></label>
    <label>目标路径<Input value={destination} onChange={(event) => setDestination(event.target.value)} /></label>
    {error && <p className="form-error">{error}</p>}
    <Button size="sm" disabled={!source || !destination} onClick={() => void upload()}><Send size={14} />开始上传</Button>
    {result !== undefined && <pre className="result-block">{pretty(result)}</pre>}
  </div>
}

function UnknownRenderer({ resource }: RendererProps) {
  return <div className="unknown-renderer"><Braces size={24} /><p>没有安装 <strong>{resource.type}</strong> 的专用 renderer。资源仍保持可发现。</p><pre>{pretty(resource)}</pre></div>
}

type RendererProps = { api: DesktopAPI; resource: ResourceDescriptor }
type RendererComponent = (props: RendererProps) => React.ReactNode

const rendererRegistry: Record<string, RendererComponent> = {
  'mfh.variable': VariableRenderer,
  'mfh.stream': EventRenderer,
  'mfh.topic': EventRenderer,
  'mfh.command': OperationRenderer,
  'mfh.file': FileRenderer,
}

export function ResourceRenderer(props: RendererProps) {
  const Renderer = rendererRegistry[props.resource.presentation?.renderer || props.resource.type] || UnknownRenderer
  const publishable = props.resource.type === 'mfh.topic' && props.resource.capabilities.some((item) => item.name === 'publish')
  const genericOperations = Renderer === UnknownRenderer ? props.resource.capabilities.filter((item) => item.input_schema && item.name !== 'open') : []
  return <div className="resource-renderer">
    {publishable && <details className="publish-box"><summary><Radio size={14} />发布到 Topic</summary><OperationRenderer {...props} capability="publish" /></details>}
    <Renderer {...props} />
    {genericOperations.map((capability) => <details className="publish-box" key={capability.name}><summary><Play size={14} />通用操作 {capability.name}</summary><OperationRenderer {...props} capability={capability.name} /></details>)}
  </div>
}

export function NodeRenderer({ node, resources }: { node: TopologyNode; resources: ResourceDescriptor[] }) {
  const counts = useMemo(() => Object.entries(resources.reduce<Record<string, number>>((all, resource) => ({ ...all, [resource.type]: (all[resource.type] || 0) + 1 }), {})), [resources])
  return <div className="node-overview">
    <div className="node-orbit"><SquareTerminal size={28} /><span>{node.node_id}</span></div>
    <div><p className="eyebrow">AUTHORITY NODE</p><h2>{node.display_name || `Node ${node.node_id}`}</h2><p>{node.parent_id ? `由 Node ${node.parent_id} 控制` : 'Authority root'} · generation {node.generation}</p></div>
    <div className="type-counts">{counts.map(([type, count]) => <span key={type}><strong>{count}</strong>{type.replace('mfh.', '')}</span>)}</div>
  </div>
}
