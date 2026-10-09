import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Info, RefreshCw, RotateCcw } from 'lucide-react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor } from '../types'
import { Button } from '../components/ui/button'
import { Textarea } from '../components/ui/input'
import { Select } from '../components/ui/select'
import { Checkbox } from '../components/ui/checkbox'
import { VariableValue } from './VariableValue'
import { selectVariableRenderer } from '../rendering/registry'
import { useVariable } from './useVariable'
import { VariableEditor } from './VariableEditor'
import { MAX_VARIABLE_BYTES, validateVariableJSON, validateVariableValue, sameValue } from './protocol'

const pretty = (value: unknown) => JSON.stringify(value, null, 2)
function friendlyTime(milliseconds: number): string {
  const date = new Date(milliseconds)
  return Number.isFinite(date.getTime()) ? date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '时间未知'
}

export function VariablePanel({ api, resource, rendererID, disabledReason, showFooter = true, autoApply = false }: { api: DesktopAPI; resource: ResourceDescriptor; rendererID?: string; disabledReason?: string; showFooter?: boolean; autoApply?: boolean }) {
  const { key, controller, state } = useVariable(api, resource)
  const { snapshot, dirty, writing, blocked, conflict } = state
  const statusID = useId()
  const definition = snapshot?.definition
  const [raw, setRaw] = useState('')
  const [inputError, setInputError] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [editorReset, setEditorReset] = useState(0)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const pendingApply = useRef<ReturnType<typeof setTimeout>>()
  const canAutoApply = useRef(false)
  const localDraft = useRef<{ controller: typeof controller; version: number; instance?: string }>()
  const confirmedBeforeWrite = useRef(snapshot?.value)
  if (!writing) confirmedBeforeWrite.current = snapshot?.value
  const cancelAutoApply = useCallback(() => {
    if (pendingApply.current !== undefined) clearTimeout(pendingApply.current)
    pendingApply.current = undefined
  }, [])
  useEffect(() => cancelAutoApply, [controller, key, autoApply, disabledReason, cancelAutoApply])
  const onValidity = useCallback((path: string, error: string) => {
    if (error) cancelAutoApply()
    setFieldErrors((current) => {
      if ((current[path] || '') === error) return current
      const next = { ...current }; if (error) next[path] = error; else delete next[path]; return next
    })
  }, [cancelAutoApply])
  useEffect(() => setRaw(pretty(state.draft) ?? ''), [key, state.draft])
  useEffect(() => setInputError(''), [key])
  const writable = resource.capabilities.some((capability) => capability.name === 'write') && !definition?.read_only
  const selection = selectVariableRenderer(resource, definition, rendererID)
  const commitAutoApply = () => {
    cancelAutoApply()
    const intent = localDraft.current
    if (!autoApply || !controller || !intent || intent.controller !== controller || disabledReason) return
    // Defer one event turn so the latest field validation has reached React.
    // Enter followed by blur cancels the first timer and submits only once.
    pendingApply.current = setTimeout(() => {
      pendingApply.current = undefined
      const current = controller.getSnapshot()
      if (!canAutoApply.current || controller.draftVersion !== intent.version || current.snapshot?.instance_id !== intent.instance) return
      if (sameValue(current.draft, current.snapshot?.value)) controller.reset()
      else void controller.apply()
    }, 0)
  }
  const edit = (value: unknown, commit = true) => {
    cancelAutoApply()
    controller?.edit(value)
    if (controller) localDraft.current = { controller, version: controller.draftVersion, instance: controller.getSnapshot().snapshot?.instance_id }
    try {
      validateVariableValue(definition!, value)
      setInputError('')
      if (commit) commitAutoApply()
    } catch (error) { setInputError(String(error)) }
  }
  const editJSON = (value: string) => {
    cancelAutoApply()
    setRaw(value)
    try { const parsed = validateVariableJSON(definition!, value); edit(parsed, false) } catch (error) { setInputError(String(error)) }
  }
  const disabled = writing || Boolean(blocked || disabledReason) || snapshot?.status === 'unavailable'
  canAutoApply.current = autoApply && !disabled && !conflict && !inputError && Object.keys(fieldErrors).length === 0
  const resetDraft = () => {
    cancelAutoApply()
    controller?.reset(); setRaw(pretty(snapshot?.value) ?? ''); setInputError(''); setFieldErrors({}); setEditorReset((value) => value + 1)
  }
  const label = definition?.title || '草稿值'
  const title = definition?.title?.trim() || resource.id.name.split('/').at(-1) || resource.id.name
  const editingValue = writable && snapshot?.status !== 'unavailable' && state.draft !== undefined && state.base?.instance_id === snapshot?.instance_id
  const displayedValue = editingValue ? state.draft : snapshot?.value
  const hasDraftChanges = dirty || Boolean(inputError) || Object.keys(fieldErrors).length > 0
  const localEdit = localDraft.current?.controller === controller && localDraft.current?.version === controller?.draftVersion
  const showUnapplied = hasDraftChanges && (!autoApply || !localEdit || Boolean(state.error || blocked || disabledReason || conflict || inputError) || Object.keys(fieldErrors).length > 0)
  const hasDraft = state.draft !== undefined
  const errors = [...new Set([disabledReason,
    snapshot?.status === 'unavailable' && blocked === '当前值不可用' ? '' : blocked,
    state.error && !blocked.includes(state.error) ? state.error : '', inputError,
  ].filter(Boolean))]
  const observation = state.loading ? '正在读取快照…' : state.observation || '尚未订阅'
  const statusTone = state.error || blocked || snapshot?.status === 'unavailable' ? 'error' : snapshot?.status === 'stale' || (state.observation && !state.live && !state.loading) ? 'idle' : state.live ? 'live' : 'idle'
  const observationFailure = !state.live && /失败|未订阅：|gap|expired|revoked|terminated/i.test(state.observation)
  const subscriptionDenied = observationFailure && /\bforbidden\b/i.test(state.observation)
  const observationNotice = subscriptionDenied
    ? snapshot ? '没有订阅权限，当前显示已读取的快照；可手动刷新。' : '没有订阅权限，无法接收实时更新。'
    : state.observation
  return <div className="value-renderer variable-panel">
    <div className="variable-header">
      <div className="variable-heading">
      <h3 className="variable-title" title={title}>{title}</h3>
      <span className={`variable-status-dot is-${statusTone}`} role="status" aria-label={observation} aria-describedby={statusID} tabIndex={0}>
        <span className="sr-only">{observation}</span>
        <span id={statusID} className="variable-status-tooltip" role="tooltip">{observation}{snapshot && <small>{friendlyTime(snapshot.observed_at_unix_ms)}</small>}</span>
      </span>
      </div>
      {Array.isArray(displayedValue) && <span className="variable-item-count">{displayedValue.length} 项</span>}
      {writing ? !autoApply && <span className="variable-draft-state">正在应用…</span> : showUnapplied && <span className="variable-draft-state">未应用</span>}
      {snapshot && !writable && <span className="variable-readonly">只读</span>}
    </div>
    {state.loading && !snapshot && <p className="renderer-state">正在读取快照…</p>}
    {observationFailure && <p className="renderer-note" role="alert">{observationNotice}</p>}
    {snapshot && definition && <>
      {!editingValue && <section className={`variable-confirmed ${writable && hasDraft ? 'is-editable' : ''}`} aria-label="已确认当前值"><small className={snapshot.status === 'stale' ? 'variable-status-warning' : 'sr-only'}>{{ ready: '可用', stale: '已过期', unavailable: '暂不可用' }[snapshot.status]}</small>{snapshot.status === 'unavailable' ? <p className="variable-status-warning">当前值不可用</p> : <VariableValue value={snapshot.value} definition={definition} rendererID={selection.selected.id} />}</section>}
      {editingValue && snapshot.status === 'stale' && <p className="variable-status-warning">已过期</p>}
      {snapshot.error && <p role="alert">{snapshot.error.code}: {snapshot.error.message}</p>}
      {selection.fallbackReason && <p className="renderer-note">{selection.fallbackReason}</p>}
      {writable && hasDraft && <fieldset className="variable-form" disabled={disabled} aria-busy={autoApply && writing || undefined}>
        <legend className="sr-only">编辑草稿</legend>
        {autoApply && writing && <span className="sr-only" role="status">正在应用</span>}
        {(definition.type === 'object' || definition.type === 'array') && <label className="variable-advanced"><Checkbox aria-label="高级 JSON 编辑" checked={advanced} onCheckedChange={setAdvanced} />高级 JSON 编辑</label>}
        {state.base?.instance_id !== snapshot.instance_id ? <p>旧实例草稿，请先重置。</p> : state.draft === undefined ? <p>没有可编辑的已确认值。</p> : advanced
          ? <label className={`schema-field${autoApply && writing ? ' is-loading-input' : ''}`}><span className="sr-only">{label} · JSON</span><Textarea aria-label={label} rows={6} maxLength={MAX_VARIABLE_BYTES} value={raw} onChange={(event) => editJSON(event.target.value)} onBlur={autoApply ? commitAutoApply : undefined} aria-busy={autoApply && writing || undefined} /></label>
          : <VariableEditor key={`${key}/${snapshot.instance_id}/${editorReset}`} definition={definition} value={state.draft as import('../generated/variable-contract.generated').JSONValue} hideLabel onChange={edit} onCommit={autoApply ? commitAutoApply : undefined} confirmedValue={autoApply ? confirmedBeforeWrite.current : undefined} busy={autoApply && writing} onValidity={onValidity} rendererID={selection.selected.id} />}

      </fieldset>}
    </>}
    {conflict && !(autoApply && writing) && <p role="alert">当前值已变化，草稿与旧 revision 冲突；请重置后编辑。</p>}
    {errors.map((message) => <p key={message} className="form-error" role="alert">{message}</p>)}
    {autoApply && !showFooter && !writing && hasDraftChanges && (state.error || conflict || blocked) && <div className="value-actions">
      <Button variant="ghost" size="sm" disabled={writing} onClick={resetDraft}>放弃修改</Button>
      {!blocked && !conflict && <Button size="sm" disabled={!canAutoApply.current || !dirty} onClick={() => void controller?.apply()}>重试应用</Button>}
    </div>}
    {showFooter && <div className="variable-footer">
      {snapshot && <details className="variable-details"><summary aria-label="技术信息"><Info size={14} aria-hidden="true" /><span>详情</span></summary><dl>
        <div><dt>资源</dt><dd>{resource.id.name}</dd></div>
        <div><dt>所有者</dt><dd>{resource.id.owner_node_id}</dd></div>
        <div><dt>类型</dt><dd>{resource.type} · v{resource.type_version}</dd></div>
        <div><dt>能力</dt><dd>{resource.capabilities.map((capability) => capability.name).join(' · ')}</dd></div>
        <div><dt>订阅状态</dt><dd>{state.observation || '尚未订阅'}</dd></div>
        <div><dt>更新时间</dt><dd>{friendlyTime(snapshot.observed_at_unix_ms)}</dd></div>
        <div><dt>revision</dt><dd>{snapshot.revision}</dd></div>
        <div><dt>instance</dt><dd>{snapshot.instance_id}</dd></div>
        <div><dt>observed_at_unix_ms</dt><dd>{snapshot.observed_at_unix_ms}</dd></div>
        {snapshot.source_time_unix_ms !== undefined && <div><dt>source_time_unix_ms</dt><dd>{snapshot.source_time_unix_ms}</dd></div>}
      </dl></details>}
    <div className="value-actions"><Button variant="ghost" size="sm" aria-label="刷新" title="刷新" disabled={state.loading} onClick={() => void controller?.refresh()}><RefreshCw size={14} aria-hidden="true" /></Button>
      {writable && hasDraft && <><Button variant="ghost" size="sm" disabled={writing} onClick={resetDraft}><RotateCcw size={13} aria-hidden="true" />重置</Button><Button size="sm" disabled={disabled || !dirty || conflict || Boolean(inputError) || Object.keys(fieldErrors).length > 0} onClick={() => void controller?.apply()}>{writing ? '应用中…' : '应用'}</Button></>}
    </div>
    </div>}
  </div>
}

export function VariableRendererSelector({ api, resource, rendererID, onRendererChange }: { api: DesktopAPI; resource: ResourceDescriptor; rendererID?: string; onRendererChange(id: string): void }) {
  const { state } = useVariable(api, resource)
  const selection = selectVariableRenderer(resource, state.snapshot?.definition, rendererID)
  if (selection.choices.length < 2) return null
  return <div className="widget-renderer-selector"><Select aria-label="显示方式" value={selection.selected.id} onValueChange={onRendererChange} options={selection.choices.map((choice) => ({ value: choice.id, label: choice.label }))} /></div>
}
