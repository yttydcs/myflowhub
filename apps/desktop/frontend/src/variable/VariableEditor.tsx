import { useEffect, useState, type KeyboardEvent } from 'react'
import { ChevronDown, Plus, X } from 'lucide-react'
import type { DataSchemaDefinition, JSONValue } from '../generated/variable-contract.generated'
import { Button } from '../components/ui/button'
import { Input, Textarea } from '../components/ui/input'
import { Select } from '../components/ui/select'
import { Switch } from '../components/ui/switch'
import { Checkbox } from '../components/ui/checkbox'
import { Slider } from '../components/ui/slider'
import { validateVariableJSON } from './protocol'

type Props = {
  definition: DataSchemaDefinition
  value: JSONValue
  onChange(value: JSONValue, commit?: boolean): void
  onCommit?(): void
  confirmedValue?: JSONValue
  busy?: boolean
  onValidity(path: string, error: string): void
  path?: string
  label?: string
  rendererID?: string
  depth?: number
  hideLabel?: boolean
}

function commitOnEnter(event: KeyboardEvent<HTMLInputElement>, onCommit?: () => void) {
  if (event.key === 'Enter' && !event.nativeEvent.isComposing && event.keyCode !== 229 && onCommit) {
    event.preventDefault()
    onCommit()
  }
}

// Defaults are used only by explicit Add/un-null actions, never to interpret a
// received value. Absence, null, false, zero and empty text stay distinct.
function initialDraft(definition: DataSchemaDefinition): JSONValue {
  if (definition.nullable || definition.type === 'null') return null
  if (definition.enum?.length) return definition.enum[0]!
  switch (definition.type) {
    case 'boolean': return false
    case 'integer':
    case 'number': return Math.max(definition.minimum ?? -Infinity, Math.min(definition.maximum ?? Infinity, 0))
    case 'string': return ''
    case 'array': return []
    case 'object': return Object.fromEntries((definition.properties || []).filter((property) => property.required).map((property) => [property.name, initialDraft(property.schema)]))
    default: throw new Error(`unsupported definition type ${definition.type}`)
  }
}

function NumberEditor({ definition, value, onChange, onValidity, path = '$', label = '草稿值', rendererID = '', hideLabel, onCommit, busy, confirmedValue }: Props) {
  const [text, setText] = useState(String(value))
  const [error, setError] = useState('')
  useEffect(() => { setText(String(value)); setError(''); onValidity(path, '') }, [value, path, onValidity])
  useEffect(() => () => onValidity(path, ''), [path, onValidity])
  const edit = (raw: string) => {
    setText(raw)
    try {
      const next = validateVariableJSON(definition, raw) as JSONValue
      onChange(next, false); setError(''); onValidity(path, '')
    } catch (failure) { const reason = String(failure); setError(reason); onValidity(path, reason) }
  }
  return <div className="schema-field variable-number-field">
    <span className={hideLabel ? 'sr-only' : undefined}>{label}</span>
    <div className="variable-number-controls">
      {rendererID.includes('slider') && <Slider aria-label={label} aria-busy={busy || undefined} min={definition.minimum} max={definition.maximum} step={definition.multiple_of ?? (definition.type === 'integer' ? 1 : 0.01)} value={busy && typeof confirmedValue === 'number' ? confirmedValue : typeof value === 'number' ? value : 0} onValueChange={(next) => edit(String(next))} onValueCommit={onCommit ? () => onCommit() : undefined} />}
      <span className={`variable-number-input${busy ? ' is-loading-input' : ''}`}><Input aria-label={rendererID.includes('slider') ? `${label} 数字输入` : label} type="number" min={definition.minimum} max={definition.maximum} step={definition.multiple_of ?? (definition.type === 'integer' ? 1 : 'any')} value={text} aria-invalid={Boolean(error)} onChange={(event) => edit(event.target.value)} onBlur={onCommit} onKeyDown={(event) => commitOnEnter(event, onCommit)} aria-busy={busy || undefined} />{definition.unit && <span>{definition.unit}</span>}</span>
    </div>
    {error && <small className="field-error" role="alert">{error}</small>}
  </div>
}

function ObjectEditor({ definition, value, onChange, onValidity, path = '$', depth = 0, onCommit, confirmedValue, busy }: Props) {
  const object = value as { [key: string]: JSONValue }
  const [newKey, setNewKey] = useState('')
  const declared = new Map((definition.properties || []).map((property) => [property.name, property]))
  const names = [...declared.keys(), ...Object.keys(object).filter((name) => !declared.has(name))]
  const remove = (name: string) => { const next = { ...object }; delete next[name]; onChange(next) }
  return <div className="structured-editor">
    {names.slice(0, 128).map((name) => {
      const property = declared.get(name)
      const schema = property?.schema || definition.additional_properties!
      const present = Object.prototype.hasOwnProperty.call(object, name)
      const label = schema.title || name
      return <section className="variable-object-field" key={name}>
        {present ? <><VariableEditor definition={schema} value={object[name]!} onChange={(next, commit) => onChange({ ...object, [name]: next }, commit)} onCommit={onCommit} confirmedValue={confirmedValue && !Array.isArray(confirmedValue) && typeof confirmedValue === 'object' ? confirmedValue[name] : undefined} busy={busy} onValidity={onValidity} path={`${path}/${JSON.stringify(name)}`} label={label} depth={depth + 1} />{!property?.required && !schema.read_only && <Button variant="ghost" size="sm" onClick={() => remove(name)}>移除 {label}</Button>}</> : <Button variant="secondary" size="sm" disabled={schema.read_only} onClick={() => onChange({ ...object, [name]: initialDraft(schema) })}>添加 {label}（当前缺失）</Button>}
      </section>
    })}
    {definition.additional_properties && <div className="variable-add-field"><Input aria-label="新增字段名" placeholder="字段名称" value={newKey} onChange={(event) => setNewKey(event.target.value)} /><Button variant="secondary" size="sm" disabled={!newKey || Object.prototype.hasOwnProperty.call(object, newKey) || Object.keys(object).length >= 128} onClick={() => { onChange({ ...object, [newKey]: initialDraft(definition.additional_properties!) }); setNewKey('') }}><Plus size={13} aria-hidden="true" />添加字段</Button></div>}
  </div>
}

function ArrayEditor({ definition, value, onChange, onValidity, path = '$', label = '列表', depth = 0, onCommit, confirmedValue, busy }: Props) {
  const array = value as JSONValue[]
  const [page, setPage] = useState(0)
  const pages = Math.max(1, Math.ceil(array.length / 10))
  const current = Math.min(page, pages - 1)
  return <div className="structured-editor">
    {array.length === 0 && <p className="variable-empty-list">暂无项目</p>}
    {array.slice(current * 10, current * 10 + 10).map((item, offset) => {
      const index = current * 10 + offset
      return <section className="variable-array-row" key={index}><span className="variable-item-index" aria-hidden="true">{index + 1}</span><VariableEditor definition={definition.items!} value={item} label={`${label}[${index}]`} hideLabel path={`${path}/${index}`} depth={depth + 1} onValidity={onValidity} onChange={(next, commit) => onChange(array.map((original, at) => at === index ? next : original), commit)} onCommit={onCommit} confirmedValue={Array.isArray(confirmedValue) ? confirmedValue[index] : undefined} busy={busy} /><Button variant="ghost" size="sm" aria-label={`移除第 ${index + 1} 项`} disabled={array.length <= (definition.min_items ?? 0)} onClick={() => onChange(array.filter((_, at) => at !== index))}><X size={14} aria-hidden="true" /></Button></section>
    })}
    {pages > 1 && <div className="value-actions"><Button size="sm" variant="ghost" disabled={current === 0} onClick={() => setPage(current - 1)}>上一页</Button><span>{current + 1} / {pages}</span><Button size="sm" variant="ghost" disabled={current === pages - 1} onClick={() => setPage(current + 1)}>下一页</Button></div>}
    <Button className="variable-add-item" size="sm" variant="ghost" disabled={array.length >= Math.min(definition.max_items ?? 1024, 1024)} onClick={() => { onChange([...array, initialDraft(definition.items!)]); setPage(Math.floor(array.length / 10)) }}><Plus size={13} aria-hidden="true" />添加一项</Button>
  </div>
}

export function VariableEditor({ definition, value, onChange, onValidity, path = '$', label = definition.title || '草稿值', rendererID = '', depth = 0, hideLabel, onCommit, confirmedValue, busy }: Props) {
  const [expanded, setExpanded] = useState(depth < 2)
  const props = { definition, value, onChange, onValidity, path, label, rendererID, depth, hideLabel, onCommit, confirmedValue, busy }
  const controlValue = onCommit && confirmedValue !== undefined ? confirmedValue : value
  let field
  if (value === null) field = definition.nullable ? null : <span>null</span>
  else if (definition.type === 'boolean') field = <label className="schema-field schema-toggle"><Switch aria-label={label} checked={controlValue === true} onCheckedChange={onChange} aria-busy={busy || undefined} /><span><strong className={hideLabel ? 'sr-only' : undefined}>{label}</strong><small>{definition.description || (controlValue === true ? '开启' : '关闭')}</small></span></label>
  else if (definition.type === 'integer' || definition.type === 'number') field = <NumberEditor {...props} />
  else if (definition.type === 'string' && definition.enum?.length && !rendererID.includes('text')) field = <label className="schema-field"><span className={hideLabel ? 'sr-only' : undefined}>{label}</span><Select aria-label={label} value={controlValue as string} onValueChange={onChange} aria-busy={busy || undefined} options={definition.enum.map((option) => ({ value: option, label: option === '' ? '空文本' : option }))} /></label>
  else if (definition.type === 'string') field = <label className={`schema-field${busy ? ' is-loading-input' : ''}${rendererID.includes('code') ? ' is-code' : ''}`}><span className={hideLabel ? 'sr-only' : undefined}>{label}</span>{!definition.sensitive && (rendererID.includes('multiline') || rendererID.includes('code')) ? <Textarea aria-label={label} rows={5} value={value as string} onChange={(event) => onChange(event.target.value, false)} onBlur={onCommit} aria-busy={busy || undefined} /> : <Input aria-label={label} value={value as string} type={definition.sensitive ? 'password' : 'text'} onKeyDown={(event) => commitOnEnter(event, onCommit)} onChange={(event) => onChange(event.target.value, false)} onBlur={onCommit} aria-busy={busy || undefined} />}</label>
  else if (depth >= 6) field = <p>已达展开深度上限；可使用高级 JSON 编辑此结构。</p>
  else if (definition.type === 'object' || definition.type === 'array') {
    const title = <>{label}{Array.isArray(value) && <span className="variable-item-count">{value.length} 项</span>}</>
    field = <>
      {depth === 0 ? !hideLabel && <div className="variable-group-title">{title}</div> : <Button className="variable-group-toggle" size="sm" variant="ghost" aria-expanded={expanded} aria-label={`${expanded ? '收起' : '展开'} ${label}`} onClick={() => setExpanded((value) => !value)}><ChevronDown size={14} aria-hidden="true" className={expanded ? '' : 'is-collapsed'} />{title}</Button>}
      {(depth === 0 || expanded) && (definition.type === 'object' ? <ObjectEditor {...props} /> : <ArrayEditor {...props} />)}
    </>
  }
  return <fieldset className="schema-field variable-field" disabled={definition.read_only}>
    {definition.nullable && <label className="variable-null-option"><Checkbox aria-label={`${label} 为 null`} checked={controlValue === null} aria-busy={busy || undefined} onCheckedChange={(checked) => onChange(checked ? null : initialDraft({ ...definition, nullable: false }))} />{label} 为 null</label>}
    {field}
  </fieldset>
}
