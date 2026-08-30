import { useEffect, useMemo, useRef, useState, type ChangeEvent, type ReactNode } from 'react'
import {
  AlertTriangle,
  Braces,
  Check,
  CirclePause,
  CirclePlay,
  Eraser,
  FileUp,
  LoaderCircle,
  Pause,
  Play,
  Radio,
  RefreshCw,
  RotateCcw,
  Search,
  Send,
  SquareTerminal,
} from 'lucide-react'
import type { DesktopAPI, OperationResult } from '../api'
import { errorText } from '../lib/utils'
import { selectResourceRenderer, type PaneDensity } from '../rendering/registry'
import {
  defaultDataValue,
  resolveSchema,
  type DataSchema,
  type DataValidationError,
  type SchemaResolution,
  validateDataValue,
} from '../rendering/schema'
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

function StateMessage({ kind, children }: { kind?: 'error' | 'loading'; children: ReactNode }) {
  return <div className={`renderer-state ${kind || ''}`} role={kind === 'error' ? 'alert' : 'status'} aria-live="polite">{kind === 'error' ? <AlertTriangle aria-hidden="true" size={17} /> : <LoaderCircle aria-hidden="true" size={17} />}{children}</div>
}

function formatScalar(value: unknown, schema?: DataSchema): string {
  if (value === null) return 'null'
  if (value === undefined) return '—'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'number') {
    if (schema?.format?.includes('unix-ms')) return new Date(value).toLocaleString()
    const precision = schema?.precision
    const formatted = precision === undefined ? String(value) : value.toFixed(precision)
    return schema?.unit ? `${formatted} ${schema.unit}` : formatted
  }
  if (typeof value === 'string' && schema?.format?.includes('unix-ms')) {
    const time = Number(value)
    if (Number.isFinite(time)) return new Date(time).toLocaleString()
  }
  return String(value)
}

function formatDateTimeLocal(value: number): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return ''
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16)
}

function summarizeCell(value: unknown): string {
  if (value === null || value === undefined || typeof value !== 'object') return formatScalar(value)
  if (Array.isArray(value)) return `${value.length} 项`
  const keys = Object.keys(value as Record<string, unknown>)
  return keys.length ? `${keys.length} 个字段` : '空对象'
}

function ObjectTable({ values, schema }: { values: unknown[]; schema?: DataSchema }) {
  const rows = values.filter((value): value is Record<string, unknown> => Boolean(value) && typeof value === 'object' && !Array.isArray(value))
  const declared = schema?.items?.properties?.map((property) => property.name) || []
  const columns = (declared.length ? declared : [...new Set(rows.flatMap((row) => Object.keys(row)))]).slice(0, 7)
  if (!columns.length) return <pre>{pretty(values)}</pre>
  return (
    <div className="structured-table-scroll">
      <table className="structured-table">
        <thead><tr>{columns.map((column) => <th key={column} scope="col">{column}</th>)}</tr></thead>
        <tbody>{rows.slice(0, 100).map((row, index) => (
          <tr key={String(row.id || row.flow_id || row.transfer_id || row.node_id || index)}>
            {columns.map((column) => <td key={column} title={typeof row[column] === 'string' ? String(row[column]) : undefined}>{summarizeCell(row[column])}</td>)}
          </tr>
        ))}</tbody>
      </table>
      {values.length > 100 && <p className="bounded-note">仅显示前 100 项，共 {values.length} 项</p>}
    </div>
  )
}

function ProgressValue({ value, schema }: { value: number; schema: DataSchema }) {
  const minimum = schema.minimum ?? 0
  const maximum = schema.maximum ?? 100
  const percent = maximum > minimum ? Math.min(100, Math.max(0, ((value - minimum) / (maximum - minimum)) * 100)) : 0
  return (
    <div className="progress-value">
      <strong>{formatScalar(value, schema)}</strong>
      <div role="progressbar" aria-valuemin={minimum} aria-valuemax={maximum} aria-valuenow={value}><span style={{ width: `${percent}%` }} /></div>
      <small>{minimum} – {maximum}</small>
    </div>
  )
}

function DataDisplay({ value, schema, density = 'normal', depth = 0, rendererID = '' }: {
  value: unknown
  schema?: DataSchema
  density?: PaneDensity
  depth?: number
  rendererID?: string
}) {
  if (schema?.write_only) return <span className="sensitive-value">不会回显的敏感字段</span>
  if ((schema?.type === 'integer' || schema?.type === 'number') && typeof value === 'number') {
    if (rendererID.includes('progress') && schema.minimum !== undefined && schema.maximum !== undefined) return <ProgressValue value={value} schema={schema} />
    return <div className="numeric-value"><strong>{formatScalar(value, schema)}</strong>{schema.unit && <small>{schema.title || '当前值'}</small>}</div>
  }
  if (schema?.type === 'boolean' && typeof value === 'boolean') {
    if (rendererID.includes('text')) return <span className="scalar-value">{formatScalar(value, schema)}</span>
    return <div className={`boolean-value ${value ? 'is-true' : 'is-false'}`}><span aria-hidden="true" /><strong>{value ? '开启' : '关闭'}</strong></div>
  }
  if (schema?.type === 'string' && typeof value === 'string') {
    if (rendererID.includes('code') || schema.format === 'json') return <pre>{value}</pre>
    if (rendererID.includes('multiline')) return <p className="multiline-value">{value}</p>
  }
  if (value === null || value === undefined || typeof value !== 'object') {
    return schema?.format === 'json' && typeof value === 'string' ? <pre>{value}</pre> : <span className="scalar-value">{formatScalar(value, schema)}</span>
  }
  if (Array.isArray(value)) {
    if (value.length === 0) return <p className="empty-copy">暂无数据</p>
    if (value.every((item) => item && typeof item === 'object' && !Array.isArray(item))) return <ObjectTable values={value} schema={schema} />
    return <ul className="structured-list">{value.slice(0, density === 'compact' ? 12 : 50).map((item, index) => <li key={index}><DataDisplay value={item} schema={schema?.items} density={density} depth={depth + 1} /></li>)}</ul>
  }
  if (depth >= 5) return <pre>{pretty(value)}</pre>
  const record = value as Record<string, unknown>
  const properties = schema?.properties || Object.keys(record).map((name) => ({ name, schema: undefined }))
  const primaryArray = properties.find((property) => Array.isArray(record[property.name]))
  const scalarProperties = properties.filter((property) => !Array.isArray(record[property.name]))
  return (
    <div className="structured-value">
      {scalarProperties.length > 0 && (
        <dl>{scalarProperties.slice(0, density === 'compact' ? 6 : 24).map((property) => (
          <div key={property.name}>
            <dt>{property.schema?.title || property.name}</dt>
            <dd><DataDisplay value={record[property.name]} schema={property.schema} density={density} depth={depth + 1} /></dd>
          </div>
        ))}</dl>
      )}
      {primaryArray && <section className="structured-section"><h4>{primaryArray.schema?.title || primaryArray.name}</h4><DataDisplay value={record[primaryArray.name]} schema={primaryArray.schema} density={density} depth={depth + 1} /></section>}
    </div>
  )
}

function errorsFor(errors: DataValidationError[], path: string): string | undefined {
  return errors.find((error) => error.path === path)?.message
}

function JSONField({ label, value, onChange, error, errorID }: { label: string; value: unknown; onChange(value: unknown): void; error?: string; errorID?: string }) {
  const [text, setText] = useState(pretty(value))
  useEffect(() => setText(pretty(value)), [value])
  return (
    <label className="schema-field schema-json-field">{label}
      <Textarea value={text} rows={5} spellCheck={false} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => {
        setText(event.target.value)
        try { onChange(JSON.parse(event.target.value)) } catch { /* keep the local invalid draft visible */ }
      }} />
      {error && <small className="field-error" id={errorID}>{error}</small>}
    </label>
  )
}

export function SchemaEditor({ schema, value, onChange, errors = [], path = '$', rendererID = '', depth = 0 }: {
  schema: DataSchema
  value: unknown
  onChange(value: unknown): void
  errors?: DataValidationError[]
  path?: string
  rendererID?: string
  depth?: number
}) {
  const label = schema.title || path.split('.').at(-1) || '值'
  const error = errorsFor(errors, path)
  const fieldID = `schema-field-${path.replace(/[^a-zA-Z0-9_-]+/g, '-')}`
  const errorID = error ? `${fieldID}-error` : undefined
  if (schema.read_only) return <div className="schema-field"><span>{label}</span><DataDisplay value={value} schema={schema} /></div>
  if (schema.type === 'boolean') {
    return <label className="schema-field schema-toggle"><input type="checkbox" checked={Boolean(value)} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => onChange(event.target.checked)} /><span><strong>{label}</strong><small>{schema.description || (value ? '开启' : '关闭')}</small></span>{error && <small className="field-error" id={errorID}>{error}</small>}</label>
  }
  if (schema.enum?.length && !rendererID.includes('text')) {
    return <label className="schema-field">{label}<select className="input" value={String(value ?? '')} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => onChange(event.target.value)}>{schema.enum.map((option) => <option key={option} value={option}>{option}</option>)}</select>{error && <small className="field-error" id={errorID}>{error}</small>}</label>
  }
  if (schema.type === 'integer' || schema.type === 'number') {
    const current = typeof value === 'number' ? value : Number(value || schema.minimum || 0)
    const step = schema.multiple_of || (schema.type === 'integer' ? 1 : 'any')
    return <div className="schema-field"><span id={fieldID}>{label}</span>
      {schema.format?.includes('unix-ms')
        ? <Input aria-labelledby={fieldID} type="datetime-local" value={Number.isFinite(current) ? formatDateTimeLocal(current) : ''} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => onChange(event.target.value === '' ? '' : new Date(event.target.value).getTime())} />
        : <>{rendererID.includes('slider') && schema.minimum !== undefined && schema.maximum !== undefined && <input className="schema-slider" type="range" min={schema.minimum} max={schema.maximum} step={step} value={current} aria-labelledby={fieldID} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => onChange(Number(event.target.value))} />}<span className="number-input-line"><Input aria-label={`${label} 数值`} type="number" min={schema.minimum} max={schema.maximum} step={step} value={Number.isFinite(current) ? current : ''} aria-invalid={Boolean(error)} aria-describedby={errorID} onChange={(event) => onChange(event.target.value === '' ? '' : Number(event.target.value))} />{schema.unit && <span>{schema.unit}</span>}</span></>}
      {schema.description && <small>{schema.description}</small>}{error && <small className="field-error" id={errorID}>{error}</small>}
    </div>
  }
  if (schema.type === 'string') {
    const multiline = !schema.sensitive && (rendererID.includes('multiline') || rendererID.includes('code') || schema.format === 'json' || (schema.max_length || 0) > 256)
    const common = { value: typeof value === 'string' ? value : '', 'aria-invalid': Boolean(error), 'aria-describedby': errorID, onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => onChange(event.target.value) }
    return <label className={`schema-field ${rendererID.includes('code') ? 'is-code' : ''}`}>{label}{multiline ? <Textarea {...common} rows={rendererID.includes('code') ? 8 : 5} spellCheck={!rendererID.includes('code')} /> : <Input {...common} type={schema.sensitive ? 'password' : 'text'} autoComplete="off" />}{schema.description && <small>{schema.description}</small>}{error && <small className="field-error" id={errorID}>{error}</small>}</label>
  }
  if (schema.type === 'array') {
    if (!schema.items || depth >= 5) return <JSONField label={label} value={value} onChange={onChange} error={error} />
    const items = Array.isArray(value) ? value : []
    const maximum = Math.min(schema.max_items ?? 64, 64)
    return <fieldset className="schema-fieldset"><legend>{label}<Badge>{items.length}</Badge></legend>{items.map((item, index) => <div className="repeatable-row" key={index}><SchemaEditor schema={schema.items!} value={item} errors={errors} path={`${path}[${index}]`} depth={depth + 1} onChange={(next) => onChange(items.map((current, candidate) => candidate === index ? next : current))} /><button type="button" onClick={() => onChange(items.filter((_, candidate) => candidate !== index))} aria-label={`移除 ${label} 第 ${index + 1} 项`}>×</button></div>)}<Button type="button" variant="secondary" size="sm" disabled={items.length >= maximum} onClick={() => onChange([...items, defaultDataValue(schema.items!)])}>添加一项</Button>{error && <small className="field-error">{error}</small>}</fieldset>
  }
  if (schema.type === 'object') {
    if ((!schema.properties || schema.properties.length === 0) && schema.additional_properties) return <JSONField label={label} value={value} onChange={onChange} error={error} />
    const record = value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {}
    return <fieldset className="schema-fieldset"><legend>{label}</legend>{(schema.properties || []).map((property) => <SchemaEditor key={property.name} schema={property.schema} value={record[property.name] ?? defaultDataValue(property.schema)} errors={errors} path={`${path}.${property.name}`} depth={depth + 1} onChange={(next) => onChange({ ...record, [property.name]: next })} />)}{error && <small className="field-error">{error}</small>}</fieldset>
  }
  return <JSONField label={label} value={value} onChange={onChange} error={error} errorID={errorID} />
}

type RendererProps = {
  api: DesktopAPI
  resource: ResourceDescriptor
  rendererID?: string
  density?: PaneDensity
}

function VariableRenderer({ api, resource, rendererID = 'mfh.variable.raw.v1', density = 'normal' }: RendererProps) {
  const [value, setValue] = useState<unknown>()
  const [revision, setRevision] = useState(0)
  const [draft, setDraft] = useState<unknown>()
  const [rawDraft, setRawDraft] = useState('')
  const [validationErrors, setValidationErrors] = useState<DataValidationError[]>([])
  const [loadError, setLoadError] = useState('')
  const [actionError, setActionError] = useState('')
  const [loading, setLoading] = useState(true)
  const [writing, setWriting] = useState(false)
  const writable = resource.capabilities.find((item) => item.name === 'write')
  const schemaID = resource.capabilities.find((item) => item.name === 'read')?.output_schema || resource.capabilities.find((item) => item.name === 'subscribe')?.event_schema
  const resolution = resolveSchema(schemaID)
  const schema = resolution.status === 'resolved' ? resolution.schema : undefined
  const refresh = async () => {
    setLoading(true); setLoadError('')
    try {
      let next: unknown
      let nextRevision = 0
      if (writable) {
        const subscriptionID = await api.subscribe(resource.id.owner_node_id, resource.id.name, 'subscribe', 5_000)
        try {
          const result = await api.poll(subscriptionID, 5_000)
          if (result.kind !== 'event') throw new Error('未收到 Variable 快照')
          const event = result.payload as ResourceEvent
          next = decodeEventPayload(event)
          nextRevision = event.revision || 0
        } finally { await api.cancel(subscriptionID) }
      } else next = await api.snapshot(resource.id.owner_node_id, resource.id.name)
      setRevision(nextRevision); setValue(next); setDraft(next); setRawDraft(pretty(next)); setValidationErrors([]); setActionError('')
    } catch (current) { setLoadError(errorText(current)) }
    finally { setLoading(false) }
  }
  const reset = () => { setDraft(value); setRawDraft(pretty(value)); setValidationErrors([]); setActionError('') }
  const write = async () => {
    setWriting(true); setActionError('')
    try {
      let next = draft
      if (rendererID.includes('raw')) next = JSON.parse(rawDraft)
      if (schema) {
        const errors = validateDataValue(schema, next)
        setValidationErrors(errors)
        if (errors.length) throw new Error(`请先修正 ${errors.length} 个字段错误`)
      }
      await api.operate(resource.id.owner_node_id, resource.id.name, 'write', writable?.input_schema || 'mfh.variable-write.v2', {
        version: 2, expected_revision: revision, value: encodeJSONBytes(next),
      })
      await refresh()
    } catch (current) { setActionError(errorText(current)) }
    finally { setWriting(false) }
  }
  useEffect(() => { void refresh() }, [resource.id.owner_node_id, resource.id.name])
  if (loading) return <StateMessage kind="loading">正在读取快照…</StateMessage>
  if (loadError) return <StateMessage kind="error">{loadError}</StateMessage>
  const raw = rendererID.includes('raw') || !schema
  return <div className="value-renderer">
    {resolution.status !== 'resolved' && <p className="renderer-note"><AlertTriangle aria-hidden="true" size={14} />{resolution.reason}</p>}
    {writable
      ? raw
        ? <label className="schema-field is-code">当前值 · revision {revision}<Textarea name="variable-value" autoComplete="off" value={rawDraft} onChange={(event) => setRawDraft(event.target.value)} rows={density === 'compact' ? 4 : 8} spellCheck={false} /></label>
        : <SchemaEditor schema={schema!} value={draft} onChange={(next) => { setDraft(next); setRawDraft(pretty(next)); setValidationErrors(validateDataValue(schema!, next)) }} errors={validationErrors} rendererID={rendererID} />
      : raw ? <pre>{pretty(value)}</pre> : <DataDisplay value={value} schema={schema} density={density} rendererID={rendererID} />}
    {actionError && <p className="form-error" role="alert" aria-live="polite">{actionError}</p>}
    <div className="value-actions"><Button variant="ghost" size="sm" onClick={() => void refresh()}><RefreshCw aria-hidden="true" size={14} />刷新</Button>{writable && <><Button variant="secondary" size="sm" disabled={writing} onClick={reset}><RotateCcw aria-hidden="true" size={14} />重置</Button><Button size="sm" disabled={writing || revision === 0} onClick={() => void write()}><Check aria-hidden="true" size={14} />{writing ? '应用中…' : '应用'}</Button></>}</div>
  </div>
}

function EventTable({ events, schema }: { events: ResourceEvent[]; schema?: DataSchema }) {
  const properties = schema?.properties?.map((property) => property.name).slice(0, 5) || []
  return <div className="structured-table-scroll"><table className="structured-table event-table"><thead><tr><th>Seq</th>{properties.map((property) => <th key={property}>{property}</th>)}</tr></thead><tbody>{events.map((event, index) => {
    const payload = decodeEventPayload(event) as Record<string, unknown>
    return <tr key={`${event.sequence || event.revision || 0}-${index}`}><td>{event.sequence || event.revision || '—'}</td>{properties.map((property) => <td key={property}>{summarizeCell(payload?.[property])}</td>)}</tr>
  })}</tbody></table></div>
}

export const MAX_VISIBLE_EVENTS = 100

export function prependBoundedEvents(current: ResourceEvent[], arrived: ResourceEvent[], limit = MAX_VISIBLE_EVENTS): ResourceEvent[] {
  if (limit <= 0) return []
  return [...arrived].reverse().concat(current).slice(0, limit)
}

function EventRenderer({ api, resource, rendererID = 'mfh.event.timeline.v1', density = 'normal' }: RendererProps) {
  const [events, setEvents] = useState<ResourceEvent[]>([])
  const [state, setState] = useState<'connecting' | 'live' | 'paused' | 'stopped'>('connecting')
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('')
  const [autoscroll, setAutoscroll] = useState(true)
  const [received, setReceived] = useState(0)
  const startedAt = useRef(Date.now())
  const paused = useRef(false)
  const contentRef = useRef<HTMLDivElement>(null)
  const pendingEvents = useRef<ResourceEvent[]>([])
  const pendingReceived = useRef(0)
  const eventFrame = useRef<number>()
  const schemaID = resource.capabilities.find((item) => item.name === 'subscribe')?.event_schema
  const resolution = resolveSchema(schemaID)
  const schema = resolution.status === 'resolved' ? resolution.schema : undefined
  useEffect(() => {
    let active = true
    let subscriptionID = 0
    const flushEvents = () => {
      eventFrame.current = undefined
      const arrived = pendingEvents.current.splice(0)
      const count = pendingReceived.current
      pendingReceived.current = 0
      if (count) setReceived((current) => current + count)
      if (arrived.length) setEvents((current) => prependBoundedEvents(current, arrived))
    }
    const scheduleEvent = (event: ResourceEvent) => {
      pendingReceived.current += 1
      if (!paused.current) pendingEvents.current.push(event)
      if (eventFrame.current === undefined) eventFrame.current = window.requestAnimationFrame(flushEvents)
    }
    startedAt.current = Date.now()
    void (async () => {
      try {
        subscriptionID = await api.subscribe(resource.id.owner_node_id, resource.id.name, 'subscribe')
        if (!active) { await api.cancel(subscriptionID); return }
        setState('live')
        while (active) {
          const result = await api.poll(subscriptionID, 1000)
          if (!active) break
          if (result.kind === 'event') {
            scheduleEvent(result.payload as ResourceEvent)
          }
          if (result.kind === 'error') throw new Error((result.payload as { message?: string })?.message || '订阅失败')
          if (result.kind === 'closed') break
        }
      } catch (current) {
        if (active) setError(errorText(current))
      } finally {
        if (active) setState('stopped')
      }
    })()
    return () => {
      active = false
      if (eventFrame.current !== undefined) window.cancelAnimationFrame(eventFrame.current)
      eventFrame.current = undefined
      pendingEvents.current = []
      pendingReceived.current = 0
      if (subscriptionID) void api.cancel(subscriptionID)
    }
  }, [api, resource.id.owner_node_id, resource.id.name])
  const visible = useMemo(() => {
    const normalized = filter.trim().toLocaleLowerCase()
    return normalized ? events.filter((event) => pretty(decodeEventPayload(event)).toLocaleLowerCase().includes(normalized)) : events
  }, [events, filter])
  const rate = received / Math.max(1, (Date.now() - startedAt.current) / 1000)
  useEffect(() => {
    if (autoscroll) contentRef.current?.scrollTo?.({ top: 0, behavior: 'smooth' })
  }, [events, autoscroll])
  const togglePause = () => {
    paused.current = !paused.current
    setState(paused.current ? 'paused' : 'live')
  }
  if (error) return <StateMessage kind="error">{error}</StateMessage>
  return <div className="event-renderer">
    <div className="event-toolbar">
      <div className="live-line" role="status" aria-live="polite"><span className={`live-dot ${state}`} aria-hidden="true" />{{ connecting: '正在建立订阅…', live: '实时订阅中', paused: '显示已暂停', stopped: '订阅已停止' }[state]}<Badge>{received} · {rate.toFixed(1)}/s</Badge></div>
      <div className="event-actions"><Button variant="ghost" size="sm" onClick={togglePause}>{paused.current ? <CirclePlay aria-hidden="true" size={14} /> : <CirclePause aria-hidden="true" size={14} />}{paused.current ? '继续' : '暂停'}</Button><Button variant="ghost" size="sm" onClick={() => { pendingEvents.current = []; setEvents([]) }}><Eraser aria-hidden="true" size={14} />清空</Button></div>
    </div>
    <label className="event-filter"><Search aria-hidden="true" size={14} /><Input aria-label="筛选事件" placeholder="筛选事件" value={filter} onChange={(event) => setFilter(event.target.value)} /><span><input type="checkbox" checked={autoscroll} onChange={(event) => setAutoscroll(event.target.checked)} />自动跟随</span></label>
    {resolution.status !== 'resolved' && <p className="renderer-note">{resolution.reason}</p>}
    {visible.some((event) => event.kind === 'gap' || event.kind === 'expired') && <p className="event-gap" role="status"><AlertTriangle aria-hidden="true" size={14} />订阅存在 gap 或已过期；当前列表可能不连续。</p>}
    <div className="event-content" ref={contentRef}>
      {visible.length === 0 ? <p className="empty-copy">等待第一条事件</p> : rendererID.includes('table')
        ? <EventTable events={visible} schema={schema} />
        : rendererID.includes('log')
          ? <pre className="event-log">{visible.map((event) => `${event.sequence || event.revision || '—'} ${pretty(decodeEventPayload(event)).replace(/\n/g, ' ')}`).join('\n')}</pre>
          : <div className="event-list">{visible.map((event, index) => <article className={event.kind === 'gap' || event.kind === 'expired' ? 'is-gap' : ''} key={`${event.sequence || event.revision || 0}-${index}`}><small>#{event.sequence || event.revision || '—'}{event.publisher_node_id ? ` · publisher ${event.publisher_node_id}/${event.publisher_sequence || 0}` : ''} · {event.schema || 'raw'}</small><DataDisplay value={decodeEventPayload(event)} schema={schema} density={density} /></article>)}</div>}
    </div>
  </div>
}

function OperationRenderer({ api, resource, capabilityName = 'invoke', rendererID = 'mfh.operation.form.v1' }: RendererProps & { capabilityName?: string }) {
  const descriptor = resource.capabilities.find((item) => item.name === capabilityName)
  const inputResolution = resolveSchema(descriptor?.input_schema)
  const schema = inputResolution.status === 'resolved' ? inputResolution.schema : undefined
  const useRaw = rendererID.includes('json') || !schema
  const [draft, setDraft] = useState<unknown>(() => schema ? defaultDataValue(schema) : {})
  const [rawDraft, setRawDraft] = useState(() => pretty(schema ? defaultDataValue(schema) : {}))
  const [validationErrors, setValidationErrors] = useState<DataValidationError[]>([])
  const [result, setResult] = useState<OperationResult>()
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    const next = schema ? defaultDataValue(schema) : {}
    setDraft(next); setRawDraft(pretty(next)); setValidationErrors([]); setResult(undefined); setError('')
  }, [descriptor?.input_schema])
  const submit = async () => {
    setBusy(true); setError('')
    try {
      const value = useRaw ? JSON.parse(rawDraft) : draft
      if (schema) {
        const errors = validateDataValue(schema, value)
        setValidationErrors(errors)
        if (errors.length) throw new Error(`请先修正 ${errors.length} 个字段错误`)
      }
      setResult(await api.operate(resource.id.owner_node_id, resource.id.name, capabilityName, descriptor?.input_schema || '', value))
    } catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }
  const outputResolution = result ? resolveSchema(result.schema || descriptor?.output_schema) : undefined
  return <div className="operation-renderer">
    {inputResolution.status !== 'resolved' && <p className="renderer-note"><AlertTriangle aria-hidden="true" size={14} />{inputResolution.reason}，使用 Advanced JSON。</p>}
    {useRaw ? <label className="schema-field is-code">输入 · {descriptor?.input_schema || 'application/json'}<Textarea name={`${capabilityName}-input`} autoComplete="off" value={rawDraft} onChange={(event) => setRawDraft(event.target.value)} rows={7} spellCheck={false} /></label> : <SchemaEditor schema={schema!} value={draft} onChange={(next) => { setDraft(next); setRawDraft(pretty(next)); setValidationErrors(validateDataValue(schema!, next)) }} errors={validationErrors} />}
    {error && <p className="form-error" role="alert" aria-live="polite">{error}</p>}
    <Button size="sm" disabled={busy} onClick={() => void submit()}><Play aria-hidden="true" size={14} />{busy ? '执行中…' : `执行 ${capabilityName}`}</Button>
    {result !== undefined && <section className="operation-result"><h4>执行结果 <Badge>{result.schema || descriptor?.output_schema || 'raw'}</Badge></h4>{outputResolution?.status === 'resolved' ? <DataDisplay value={result.payload} schema={outputResolution.schema} /> : <pre className="result-block">{pretty(result.payload)}</pre>}</section>}
  </div>
}

function FileRenderer({ api, resource }: RendererProps) {
  const [source, setSource] = useState('')
  const [destination, setDestination] = useState('')
  const [result, setResult] = useState<unknown>()
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const choose = async () => {
    setError('')
    try {
      const selected = await api.pickFile()
      if (!selected) return
      setSource(selected)
      if (!destination) setDestination(selected.split(/[\\/]/).at(-1) || '')
    } catch (current) { setError(errorText(current)) }
  }
  const upload = async () => {
    setBusy(true); setError(''); setResult(undefined)
    try { setResult(await api.uploadFile(resource.id.owner_node_id, source, destination, 'application/octet-stream')) }
    catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }
  return <div className="file-renderer">
    <FileUp aria-hidden="true" size={26} />
    <div><strong>通过有界 Resource session 上传</strong><p>文件块不占普通控制消息队列；目标路径仍由 owner 校验。</p></div>
    <label className="schema-field">本地文件<div className="file-picker-line"><Input name="upload-source" autoComplete="off" spellCheck={false} value={source} readOnly placeholder="尚未选择文件" /><Button type="button" variant="secondary" onClick={() => void choose()}>选择文件</Button></div></label>
    <label className="schema-field">目标相对路径<Input name="upload-destination" autoComplete="off" spellCheck={false} value={destination} onChange={(event) => setDestination(event.target.value)} /></label>
    {busy && <div className="transfer-pending" role="status"><LoaderCircle aria-hidden="true" size={15} />正在建立 session 并传输…</div>}
    {error && <p className="form-error" role="alert" aria-live="polite">{error}</p>}
    <Button size="sm" disabled={busy || !source || !destination} onClick={() => void upload()}><Send aria-hidden="true" size={14} />{busy ? '上传中…' : '开始上传'}</Button>
    {result !== undefined && <section className="operation-result"><h4>传输结果</h4><DataDisplay value={result} /></section>}
  </div>
}

function UnknownRenderer({ resource }: RendererProps) {
  return <div className="unknown-renderer"><Braces aria-hidden="true" size={24} /><p>没有安装 <strong>{resource.type}</strong> 的专用 renderer。资源仍保持可发现。</p><pre>{pretty(resource)}</pre></div>
}

export function ResourceRenderer({ api, resource, rendererID, density = 'normal' }: RendererProps) {
  const selection = selectResourceRenderer(resource, rendererID)
  const selectedID = selection.selected.id
  const publishable = resource.type === 'mfh.topic' && resource.capabilities.some((item) => item.name === 'publish')
  const knownType = ['mfh.variable', 'mfh.stream', 'mfh.topic', 'mfh.command', 'mfh.file'].includes(resource.type)
  const genericOperations = knownType ? [] : resource.capabilities.filter((item) => item.input_schema && item.name !== 'open')
  return <div className={`resource-renderer density-${density}`}>
    {selection.fallbackReason && <p className="renderer-note"><AlertTriangle aria-hidden="true" size={14} />{selection.fallbackReason}</p>}
    {publishable && <details className="publish-box"><summary><Radio aria-hidden="true" size={14} />发布到 Topic</summary><OperationRenderer api={api} resource={resource} capabilityName="publish" rendererID="mfh.operation.form.v1" density={density} /></details>}
    {resource.type === 'mfh.variable' && <VariableRenderer api={api} resource={resource} rendererID={selectedID} density={density} />}
    {(resource.type === 'mfh.stream' || resource.type === 'mfh.topic') && <EventRenderer api={api} resource={resource} rendererID={selectedID} density={density} />}
    {resource.type === 'mfh.command' && <OperationRenderer api={api} resource={resource} rendererID={selectedID} density={density} />}
    {resource.type === 'mfh.file' && <FileRenderer api={api} resource={resource} rendererID={selectedID} density={density} />}
    {!knownType && <UnknownRenderer api={api} resource={resource} rendererID={selectedID} density={density} />}
    {genericOperations.map((descriptor) => <details className="publish-box" key={descriptor.name}><summary><Play aria-hidden="true" size={14} />通用操作 {descriptor.name}</summary><OperationRenderer api={api} resource={resource} capabilityName={descriptor.name} rendererID="mfh.operation.form.v1" density={density} /></details>)}
  </div>
}

export function ResourceRendererSelector({ resource, rendererID, onRendererChange }: Pick<RendererProps, 'resource' | 'rendererID'> & { onRendererChange(rendererID: string): void }) {
  const selection = selectResourceRenderer(resource, rendererID)
  if (selection.choices.length < 2) return null
  return (
    <label className="widget-renderer-selector">
      <span className="sr-only">显示方式</span>
      <select aria-label="显示方式" value={selection.selected.id} onChange={(event) => onRendererChange(event.target.value)}>
        {selection.choices.map((choice) => <option key={choice.id} value={choice.id}>{choice.label}</option>)}
      </select>
    </label>
  )
}

export function NodeRenderer({ node, resources }: { node: TopologyNode; resources: ResourceDescriptor[] }) {
  const counts = useMemo(() => {
    const byType = new Map<string, number>()
    for (const resource of resources) byType.set(resource.type, (byType.get(resource.type) || 0) + 1)
    return [...byType.entries()]
  }, [resources])
  return <div className="node-overview">
    <div className="node-orbit"><SquareTerminal aria-hidden="true" size={28} /><span>{node.node_id}</span></div>
    <dl className="inspector-facts">
      <div><dt>Node ID</dt><dd>{node.node_id}</dd></div>
      <div><dt>角色</dt><dd>{node.role}</dd></div>
      <div><dt>父 Node</dt><dd>{node.parent_id || 'Authority root'}</dd></div>
      <div><dt>Generation</dt><dd>{node.generation}</dd></div>
    </dl>
    <div className="type-counts">{counts.map(([type, count]) => <span key={type}><strong>{count}</strong>{type.replace('mfh.', '')}</span>)}</div>
  </div>
}
