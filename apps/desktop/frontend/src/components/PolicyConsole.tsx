import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { KeyRound, RefreshCw, ShieldAlert, Trash2 } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { errorText } from '../lib/utils'
import { Button } from './ui/button'
import { Input } from './ui/input'

type CollectionMember = { key: string }
type CollectionPage = { revision: number; members: CollectionMember[]; next_cursor?: string }
type ResourceSelector = { kind: 'exact' | 'prefix' | 'all'; value?: string }
type CapabilitySelector = { kind: 'exact' | 'all'; values?: string[] }
type PolicyRule = { resource: ResourceSelector; capability: CapabilitySelector }
type PolicyDefinition = { version: 1; id: string; label: string; revision: number; immutable?: boolean; rules: PolicyRule[] }
type PolicyScope = { kind: 'owner' | 'subtree' | 'authority-domain'; node_id: string }
type PolicyBinding = {
  version: 1
  binding_id: string
  subject: string
  definition_id: string
  scope: PolicyScope
  created_by: string
  created_at_unix_ms: number
  expires_at_unix_ms?: number
}
type PolicyGrant = { version: 1; subject: string; capability: string; resource_node: string; resource_name: string }
type PolicyEvaluation = {
  allowed: boolean
  source: 'none' | 'exact-grant' | 'binding'
  policy_generation: number
  topology_epoch?: number
  binding_id?: string
  definition_id?: string
  definition_revision?: number
  rule_index?: number
  scope?: PolicyScope
}

const resources = {
  definitions: 'system/policy/definitions',
  bindings: 'system/policy/bindings',
  grants: 'system/policy/grants',
}

const schemas = {
  list: 'mfh.collection.list-request.v1',
  member: 'mfh.collection.member-request.v1',
  definitionPut: 'mfh.policy.definition-put.v1',
  definitionDelete: 'mfh.policy.definition-delete.v1',
  bindingCreate: 'mfh.policy.binding-create.v1',
  bindingRevoke: 'mfh.policy.binding-revoke.v1',
  grant: 'mfh.policy.grant.v1',
  evaluate: 'mfh.policy.evaluate-request.v1',
}

const initialRules = JSON.stringify([{
  resource: { kind: 'prefix', value: 'metrics' },
  capability: { kind: 'exact', values: ['read', 'subscribe'] },
}], null, 2)

function randomBindingID(): string {
  const value = new Uint8Array(16)
  crypto.getRandomValues(value)
  return Array.from(value, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

function shortID(value: string): string {
  return value.length > 20 ? `${value.slice(0, 9)}…${value.slice(-7)}` : value
}

export function PolicyConsole({
  api,
  authorityNodeID,
  currentNodeID,
}: {
  api: DesktopAPI
  authorityNodeID: string
  currentNodeID: string
}) {
  const [definitions, setDefinitions] = useState<PolicyDefinition[]>([])
  const [bindings, setBindings] = useState<PolicyBinding[]>([])
  const [grants, setGrants] = useState<PolicyGrant[]>([])
  const [generation, setGeneration] = useState(0)
  const [loading, setLoading] = useState(true)
  const [acting, setActing] = useState(false)
  const [error, setError] = useState('')
  const [definitionID, setDefinitionID] = useState('')
  const [definitionLabel, setDefinitionLabel] = useState('')
  const [expectedRevision, setExpectedRevision] = useState(0)
  const [rulesJSON, setRulesJSON] = useState(initialRules)
  const [bindingSubject, setBindingSubject] = useState(currentNodeID)
  const [bindingDefinition, setBindingDefinition] = useState('')
  const [scopeKind, setScopeKind] = useState<PolicyScope['kind']>('authority-domain')
  const [scopeNode, setScopeNode] = useState(authorityNodeID)
  const [grantSubject, setGrantSubject] = useState(currentNodeID)
  const [grantCapability, setGrantCapability] = useState('read')
  const [grantOwner, setGrantOwner] = useState(authorityNodeID)
  const [grantResource, setGrantResource] = useState('system/health')
  const [evaluateSubject, setEvaluateSubject] = useState(currentNodeID)
  const [evaluateCapability, setEvaluateCapability] = useState('read')
  const [evaluateOwner, setEvaluateOwner] = useState(authorityNodeID)
  const [evaluateResource, setEvaluateResource] = useState('system/health')
  const [evaluation, setEvaluation] = useState<PolicyEvaluation | null>(null)

  const operate = useCallback(async (name: string, capability: string, schema: string, input: unknown) => (
    await api.operate(authorityNodeID, name, capability, schema, input)
  ).payload, [api, authorityNodeID])

  const listRecords = useCallback(async <T,>(name: string): Promise<{ revision: number; items: T[] }> => {
    const members: CollectionMember[] = []
    let cursor = ''
    let revision = 0
    for (let pageIndex = 0; pageIndex < 1_000; pageIndex += 1) {
      const page = await operate(name, 'list', schemas.list, { version: 1, limit: 256, cursor }) as CollectionPage
      if (revision && page.revision !== revision) throw new Error(`${name} changed while pages were being read; refresh and retry`)
      revision = page.revision
      members.push(...(page.members || []))
      if (!page.next_cursor) {
        const items = await Promise.all(members.map(async (member) => (
          await operate(name, 'get', schemas.member, { version: 1, key: member.key }) as T
        )))
        return { revision, items }
      }
      if (page.next_cursor === cursor) throw new Error(`${name} returned a repeated cursor`)
      cursor = page.next_cursor
    }
    throw new Error(`${name} exceeded the pagination safety limit`)
  }, [operate])

  const refresh = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [definitionResult, bindingResult, grantResult] = await Promise.all([
        listRecords<PolicyDefinition>(resources.definitions),
        listRecords<PolicyBinding>(resources.bindings),
        listRecords<PolicyGrant>(resources.grants),
      ])
      const revisions = [definitionResult.revision, bindingResult.revision, grantResult.revision]
      if (revisions.some((value) => value !== revisions[0])) throw new Error('Policy changed while the console was loading; refresh and retry')
      setDefinitions(definitionResult.items)
      setBindings(bindingResult.items)
      setGrants(grantResult.items)
      setGeneration(revisions[0] || 0)
      setBindingDefinition((current) => current || definitionResult.items[0]?.id || '')
    } catch (current) {
      setError(errorText(current))
    } finally {
      setLoading(false)
    }
  }, [listRecords])

  useEffect(() => { void refresh() }, [refresh])
  useEffect(() => {
    setBindingSubject((current) => current || currentNodeID)
    setGrantSubject((current) => current || currentNodeID)
    setEvaluateSubject((current) => current || currentNodeID)
  }, [currentNodeID])

  async function act(operation: () => Promise<void>) {
    setActing(true)
    setError('')
    try {
      await operation()
      await refresh()
    } catch (current) {
      setError(errorText(current))
    } finally {
      setActing(false)
    }
  }

  async function saveDefinition(event: FormEvent) {
    event.preventDefault()
    await act(async () => {
      const rules = JSON.parse(rulesJSON) as PolicyRule[]
      if (rules.some((rule) => rule.resource.kind === 'all' || rule.capability.kind === 'all')
        && !window.confirm('此 Definition 包含 all 选择器，将自动覆盖未来加入的匹配资源或 Capability。继续吗？')) return
      const capability = expectedRevision ? 'update' : 'create'
      await operate(resources.definitions, capability, schemas.definitionPut, {
        version: 1,
        id: definitionID.trim(),
        label: definitionLabel.trim(),
        expected_revision: expectedRevision,
        rules,
      })
      setDefinitionID('')
      setDefinitionLabel('')
      setExpectedRevision(0)
      setRulesJSON(initialRules)
    })
  }

  function editDefinition(definition: PolicyDefinition) {
    if (definition.immutable) return
    setDefinitionID(definition.id)
    setDefinitionLabel(definition.label)
    setExpectedRevision(definition.revision)
    setRulesJSON(JSON.stringify(definition.rules, null, 2))
  }

  async function removeDefinition(definition: PolicyDefinition) {
    if (!window.confirm(`删除 Definition “${definition.label}”？仍被 Binding 引用时 Authority 会拒绝。`)) return
    await act(async () => {
      await operate(resources.definitions, 'delete', schemas.definitionDelete, { version: 1, id: definition.id, expected_revision: definition.revision })
    })
  }

  async function createBinding(event: FormEvent) {
    event.preventDefault()
    if (bindingDefinition === 'superadmin'
      && !window.confirm('superadmin 会授予作用域内所有当前及未来 Resource/Capability。确认创建此持久 Binding？')) return
    await act(async () => {
      await operate(resources.bindings, 'create', schemas.bindingCreate, {
        version: 1,
        binding_id: randomBindingID(),
        subject: bindingSubject.trim(),
        definition_id: bindingDefinition,
        scope: { kind: scopeKind, node_id: scopeNode.trim() },
      })
    })
  }

  async function revokeBinding(binding: PolicyBinding) {
    if (!window.confirm(`撤销 Subject ${binding.subject} 的 ${binding.definition_id} Binding？活动会话可能随策略代次失效。`)) return
    await act(async () => {
      await operate(resources.bindings, 'revoke', schemas.bindingRevoke, { version: 1, binding_id: binding.binding_id })
    })
  }

  const grantInput = useMemo<PolicyGrant>(() => ({
    version: 1,
    subject: grantSubject.trim(),
    capability: grantCapability.trim(),
    resource_node: grantOwner.trim(),
    resource_name: grantResource.trim(),
  }), [grantCapability, grantOwner, grantResource, grantSubject])

  async function createGrant(event: FormEvent) {
    event.preventDefault()
    await act(async () => { await operate(resources.grants, 'create', schemas.grant, grantInput) })
  }

  async function revokeGrant(grant: PolicyGrant) {
    if (!window.confirm(`撤销 ${grant.subject} 对 ${grant.resource_node}/${grant.resource_name} 的 ${grant.capability}？`)) return
    await act(async () => { await operate(resources.grants, 'revoke', schemas.grant, grant) })
  }

  async function evaluate(event: FormEvent) {
    event.preventDefault()
    setActing(true)
    setError('')
    try {
      const result = await operate(resources.bindings, 'evaluate', schemas.evaluate, {
        version: 1,
        subject: evaluateSubject.trim(),
        capability: evaluateCapability.trim(),
        resource_node: evaluateOwner.trim(),
        resource_name: evaluateResource.trim(),
      }) as PolicyEvaluation
      setEvaluation(result)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setActing(false)
    }
  }

  return (
    <div className="settings-section policy-console">
      <header>
        <div><h2>权限策略</h2><p>Authority Node {authorityNodeID} · generation {generation || '—'}</p></div>
        <Button variant="secondary" size="sm" disabled={loading || acting} onClick={() => void refresh()}><RefreshCw aria-hidden="true" size={13} />刷新</Button>
      </header>
      <p className="policy-neutrality"><KeyRound aria-hidden="true" size={14} />权限绑定到精确 Subject Node ID；Desktop、Metrics 与 Agent Gateway 都不会因产品类型获得特权。</p>
      {error && <p className="settings-error" role="alert">Authority 策略操作失败：{error}</p>}
      {loading ? <p className="admission-state">正在读取 Definitions、Bindings 与精确 Grants…</p> : (
        <>
          <section className="admission-block">
            <div className="admission-block-heading"><div><h3>Definitions</h3><p>可复用规则集合；owner 范围只出现在 Binding 中。</p></div></div>
            <div className="admission-records">
              {definitions.map((definition) => (
                <article key={definition.id}>
                  <div><strong>{definition.label}</strong><span className={`admission-badge ${definition.immutable ? 'active' : ''}`}>{definition.immutable ? 'immutable' : `rev ${definition.revision}`}</span></div>
                  <p><code>{definition.id}</code> · {definition.rules.length} 条规则</p>
                  <div className="record-actions">
                    {!definition.immutable && <Button variant="secondary" size="sm" onClick={() => editDefinition(definition)}>编辑</Button>}
                    {!definition.immutable && <Button variant="secondary" size="sm" onClick={() => void removeDefinition(definition)}><Trash2 aria-hidden="true" size={12} />删除</Button>}
                  </div>
                </article>
              ))}
            </div>
            <form className="policy-form" onSubmit={(event) => void saveDefinition(event)}>
              <label>Definition ID<Input required disabled={expectedRevision > 0} pattern="[A-Za-z0-9._-]{1,128}" value={definitionID} onChange={(event) => setDefinitionID(event.target.value)} /></label>
              <label>显示名称<Input required value={definitionLabel} onChange={(event) => setDefinitionLabel(event.target.value)} /></label>
              <label className="policy-rules">规则 JSON<textarea required value={rulesJSON} onChange={(event) => setRulesJSON(event.target.value)} /></label>
              <div className="record-actions"><Button type="submit" disabled={acting}>{expectedRevision ? '更新 Definition' : '创建 Definition'}</Button>{expectedRevision > 0 && <Button type="button" variant="secondary" onClick={() => { setExpectedRevision(0); setDefinitionID(''); setDefinitionLabel(''); setRulesJSON(initialRules) }}>取消编辑</Button>}</div>
            </form>
          </section>

          <section className="admission-block">
            <div className="admission-block-heading"><div><h3>Subject Bindings</h3><p>Definition + 精确 Subject + owner 作用域；持久化于 Authority。</p></div></div>
            <div className="admission-records">
              {bindings.length === 0 && <p className="admission-state">暂无 Binding。</p>}
              {bindings.map((binding) => (
                <article key={binding.binding_id}>
                  <div><strong>Node {binding.subject} · {binding.definition_id}</strong><span className="admission-badge active">{binding.scope.kind}</span></div>
                  <p>scope {binding.scope.node_id} · Binding {shortID(binding.binding_id)} · created by {binding.created_by}</p>
                  <div className="record-actions"><Button variant="secondary" size="sm" disabled={acting} onClick={() => void revokeBinding(binding)}><Trash2 aria-hidden="true" size={12} />撤销</Button></div>
                </article>
              ))}
            </div>
            <form className="policy-form policy-grid" onSubmit={(event) => void createBinding(event)}>
              <label>Subject Node ID<Input required inputMode="numeric" pattern="[1-9][0-9]*" value={bindingSubject} onChange={(event) => setBindingSubject(event.target.value)} /></label>
              <label>Definition<select value={bindingDefinition} onChange={(event) => setBindingDefinition(event.target.value)}>{definitions.map((definition) => <option key={definition.id} value={definition.id}>{definition.label}</option>)}</select></label>
              <label>作用域<select value={scopeKind} onChange={(event) => setScopeKind(event.target.value as PolicyScope['kind'])}><option value="owner">单个 Owner</option><option value="subtree">子树</option><option value="authority-domain">Authority domain</option></select></label>
              <label>作用域 Node ID<Input required inputMode="numeric" pattern="[1-9][0-9]*" value={scopeNode} onChange={(event) => setScopeNode(event.target.value)} /></label>
              <Button type="submit" disabled={acting || !bindingDefinition}>创建 Binding</Button>
            </form>
          </section>

          <section className="admission-block">
            <div className="admission-block-heading"><div><h3>精确 Grants</h3><p>兼容最小权限的 Subject + Capability + Resource 元组。</p></div></div>
            <div className="admission-records">
              {grants.length === 0 && <p className="admission-state">暂无精确 Grant。</p>}
              {grants.map((grant) => (
                <article key={`${grant.subject}:${grant.capability}:${grant.resource_node}:${grant.resource_name}`}>
                  <div><strong>Node {grant.subject}</strong><span className="admission-badge">{grant.capability}</span></div>
                  <p>{grant.resource_node} / {grant.resource_name}</p>
                  <div className="record-actions"><Button variant="secondary" size="sm" disabled={acting} onClick={() => void revokeGrant(grant)}><Trash2 aria-hidden="true" size={12} />撤销</Button></div>
                </article>
              ))}
            </div>
            <form className="policy-form policy-grid" onSubmit={(event) => void createGrant(event)}>
              <label>Subject<Input required value={grantSubject} onChange={(event) => setGrantSubject(event.target.value)} /></label>
              <label>Capability<Input required value={grantCapability} onChange={(event) => setGrantCapability(event.target.value)} /></label>
              <label>Resource Owner<Input required value={grantOwner} onChange={(event) => setGrantOwner(event.target.value)} /></label>
              <label>Resource Name<Input required value={grantResource} onChange={(event) => setGrantResource(event.target.value)} /></label>
              <Button type="submit" disabled={acting}>创建精确 Grant</Button>
            </form>
          </section>

          <section className="admission-block">
            <div className="admission-block-heading"><div><h3>Effective Preview</h3><p>由 Authority 按当前 policy generation 与 topology epoch 解释。</p></div></div>
            <form className="policy-form policy-grid" onSubmit={(event) => void evaluate(event)}>
              <label>Subject<Input required value={evaluateSubject} onChange={(event) => setEvaluateSubject(event.target.value)} /></label>
              <label>Capability<Input required value={evaluateCapability} onChange={(event) => setEvaluateCapability(event.target.value)} /></label>
              <label>Resource Owner<Input required value={evaluateOwner} onChange={(event) => setEvaluateOwner(event.target.value)} /></label>
              <label>Resource Name<Input required value={evaluateResource} onChange={(event) => setEvaluateResource(event.target.value)} /></label>
              <Button type="submit" disabled={acting}>评估</Button>
            </form>
            {evaluation && <div className={`policy-evaluation ${evaluation.allowed ? 'is-allowed' : 'is-denied'}`} role="status"><ShieldAlert aria-hidden="true" size={15} /><div><strong>{evaluation.allowed ? '允许' : '拒绝'} · {evaluation.source}</strong><p>generation {evaluation.policy_generation}{evaluation.topology_epoch ? ` · topology ${evaluation.topology_epoch}` : ''}{evaluation.binding_id ? ` · ${shortID(evaluation.binding_id)}` : ''}</p></div></div>}
          </section>
        </>
      )}
    </div>
  )
}
