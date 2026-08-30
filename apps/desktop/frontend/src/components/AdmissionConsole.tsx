import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Check, Copy, RefreshCw, ShieldCheck, ShieldX } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { errorText } from '../lib/utils'
import { Button } from './ui/button'
import { Input } from './ui/input'

type PermitRecord = {
  permit_id: string
  device_public_key_fingerprint: string
  target_node_id: string
  allow_descendants?: boolean
  admission_profile: string
  expires_at_unix_ms: number
  status: string
}

type RequestRecord = {
  request_id: string
  device_public_key_fingerprint: string
  parent_node_id: string
  created_at_unix_ms: number
  expires_at_unix_ms: number
  status: string
  reason?: string
}

type EnrollmentRecord = {
  enrollment_id: string
  node_id: string
  device_public_key_fingerprint: string
  parent_node_id: string
  admission_profile: string
  status: string
  reason?: string
}

type AdmissionList<T> = { authority_epoch: number; items: T[]; next_cursor?: string }

const schemas = {
  list: 'mfh.admission.list.v1',
  issue: 'mfh.admission.issue-permit.v1',
  revokePermit: 'mfh.admission.revoke-permit.v1',
  decision: 'mfh.admission.decision.v1',
  revokeEnrollment: 'mfh.admission.revoke-enrollment.v1',
}

function nextRequestID(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('')
}

function shortID(value: string): string {
  return value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-6)}` : value
}

function dateTime(value: number): string {
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

async function deviceFingerprint(value: string): Promise<string> {
  const normalized = value.trim()
  if (/^[0-9a-f]{64}$/i.test(normalized)) return normalized.toLowerCase()
  if (!/^[A-Za-z0-9+/]{43}$/.test(normalized)) throw new Error('设备身份必须是 raw-base64 Ed25519 公钥或 64 位十六进制 SHA-256 指纹')
  const decoded = Uint8Array.from(atob(`${normalized}=`), (character) => character.charCodeAt(0))
  if (decoded.length !== 32) throw new Error('设备公钥必须包含 32 字节 Ed25519 公钥')
  const digest = new Uint8Array(await crypto.subtle.digest('SHA-256', decoded))
  return Array.from(digest, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

export function AdmissionConsole({ api, authorityNodeID }: { api: DesktopAPI; authorityNodeID: string }) {
  const [permits, setPermits] = useState<PermitRecord[]>([])
  const [requests, setRequests] = useState<RequestRecord[]>([])
  const [enrollments, setEnrollments] = useState<EnrollmentRecord[]>([])
  const [epoch, setEpoch] = useState(0)
  const [loading, setLoading] = useState(true)
  const [acting, setActing] = useState(false)
  const [error, setError] = useState('')
  const [fingerprint, setFingerprint] = useState('')
  const [targetNodeID, setTargetNodeID] = useState('')
  const [admissionProfile, setAdmissionProfile] = useState('member')
  const [ttlHours, setTTLHours] = useState('24')
  const [allowDescendants, setAllowDescendants] = useState(false)
  const [issuedPermit, setIssuedPermit] = useState('')
  const [copyStatus, setCopyStatus] = useState('')

  const operate = useCallback(async (name: string, schema: string, input: unknown) => (
    await api.operate(authorityNodeID, name, 'invoke', schema, input)
  ).payload, [api, authorityNodeID])

  const refresh = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const listAll = async <T,>(name: string): Promise<AdmissionList<T>> => {
        const items: T[] = []
        let cursor = ''
        let authorityEpoch = 0
        for (let page = 0; page < 1_000; page += 1) {
          const result = await operate(name, schemas.list, { version: 1, limit: 256, cursor }) as AdmissionList<T>
          if (authorityEpoch !== 0 && result.authority_epoch !== authorityEpoch) throw new Error(`Authority ${name} changed while pages were being read; refresh and retry`)
          authorityEpoch = result.authority_epoch || authorityEpoch
          items.push(...(result.items || []))
          if (!result.next_cursor) return { authority_epoch: authorityEpoch, items }
          if (result.next_cursor === cursor) throw new Error(`Authority ${name} returned a repeated cursor`)
          cursor = result.next_cursor
        }
        throw new Error(`Authority ${name} exceeded the pagination safety limit`)
      }
      const [permitResult, requestResult, enrollmentResult] = await Promise.all([
        listAll<PermitRecord>('system/admission/list-permits'),
        listAll<RequestRecord>('system/admission/list-requests'),
        listAll<EnrollmentRecord>('system/admission/list-enrollments'),
      ])
	  const epochs = [permitResult.authority_epoch, requestResult.authority_epoch, enrollmentResult.authority_epoch].filter((value) => value !== 0)
	  if (epochs.some((value) => value !== epochs[0])) throw new Error('Authority changed while the admission view was being read; refresh and retry')
      setPermits(permitResult.items || [])
      setRequests(requestResult.items || [])
      setEnrollments(enrollmentResult.items || [])
	  setEpoch(epochs[0] || 0)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setLoading(false)
    }
  }, [operate])

  useEffect(() => { void refresh() }, [refresh])

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

  async function issue(event: FormEvent) {
    event.preventDefault()
    const hours = Number(ttlHours)
    await act(async () => {
      const normalizedFingerprint = await deviceFingerprint(fingerprint)
      const permit = await operate('system/admission/issue', schemas.issue, {
        version: 1,
        request_id: nextRequestID(),
        device_public_key_fingerprint: normalizedFingerprint,
        target_node_id: targetNodeID.trim(),
        allow_descendants: allowDescendants,
        admission_profile: admissionProfile.trim(),
        ttl_ms: hours * 60 * 60 * 1000,
      })
      setIssuedPermit(JSON.stringify(permit))
      setCopyStatus('')
    })
  }

  async function copyPermit() {
    try {
      await navigator.clipboard.writeText(issuedPermit)
      setCopyStatus('已复制；请通过安全的带外渠道交给目标设备。')
    } catch {
      setCopyStatus('复制失败，请手动选择 Permit。')
    }
  }

  return (
    <div className="settings-section admission-console">
      <header>
        <div><h2>准入管理</h2><p>Authority Node {authorityNodeID} · epoch {epoch || '—'}</p></div>
        <Button variant="secondary" size="sm" disabled={loading || acting} onClick={() => void refresh()}><RefreshCw aria-hidden="true" size={13} />刷新</Button>
      </header>

      {error && <p className="settings-error" role="alert">无法读取或修改 Authority：{error}</p>}
      {loading
        ? <p className="admission-state">正在读取集中准入状态…</p>
        : (
          <>
            <section className="admission-block" aria-labelledby="issue-permit-title">
              <div className="admission-block-heading"><div><h3 id="issue-permit-title">签发一次性 Permit</h3><p>Permit 绑定设备公钥指纹和目标父节点，不包含预分配 Node ID。</p></div></div>
              <form className="admission-issue-form" onSubmit={(event) => void issue(event)}>
                <label>设备公钥或 SHA-256 指纹<Input required value={fingerprint} onChange={(event) => setFingerprint(event.target.value)} placeholder="raw-base64 公钥或 64 位十六进制指纹" /></label>
                <label>目标父 Node ID<Input required inputMode="numeric" pattern="[1-9][0-9]*" value={targetNodeID} onChange={(event) => setTargetNodeID(event.target.value)} /></label>
                <label>准入 Profile<Input required pattern="[A-Za-z0-9._-]{1,128}" value={admissionProfile} onChange={(event) => setAdmissionProfile(event.target.value)} /></label>
                <label>有效期（小时）<Input required type="number" min="1" max="8760" value={ttlHours} onChange={(event) => setTTLHours(event.target.value)} /></label>
                <label className="check-row"><input type="checkbox" checked={allowDescendants} onChange={(event) => setAllowDescendants(event.target.checked)} />允许目标节点的后代父节点使用</label>
                <Button type="submit" disabled={acting}><ShieldCheck aria-hidden="true" size={13} />签发 Permit</Button>
              </form>
              {issuedPermit && (
                <div className="issued-permit">
                  <div><strong>新 Permit</strong><Button variant="secondary" size="sm" onClick={() => void copyPermit()}><Copy aria-hidden="true" size={12} />复制</Button></div>
                  <code>{issuedPermit}</code>
                  {copyStatus && <p aria-live="polite">{copyStatus}</p>}
                </div>
              )}
            </section>

            <section className="admission-block" aria-labelledby="pending-title">
              <div className="admission-block-heading"><div><h3 id="pending-title">待审批请求</h3><p>{requests.filter((item) => item.status === 'pending').length} 个待处理</p></div></div>
              <div className="admission-records">
                {requests.length === 0 && <p className="admission-state">暂无注册申请。</p>}
                {requests.map((request) => (
                  <article key={request.request_id}>
                    <div><strong>{shortID(request.device_public_key_fingerprint)}</strong><span className={`admission-badge ${request.status}`}>{request.status}</span></div>
                    <p>父 Node {request.parent_node_id} · {dateTime(request.created_at_unix_ms)} · 请求 {shortID(request.request_id)}</p>
                    {request.reason && <p>{request.reason}</p>}
                    {request.status === 'pending' && <div className="record-actions">
                      <Button size="sm" disabled={acting} onClick={() => void act(async () => { await operate('system/admission/approve', schemas.decision, { version: 1, request_id: nextRequestID(), enrollment_request_id: request.request_id, admission_profile: 'member' }) })}><Check aria-hidden="true" size={12} />批准</Button>
                      <Button variant="secondary" size="sm" disabled={acting} onClick={() => {
                        const reason = window.prompt('拒绝原因')?.trim()
                        if (reason) void act(async () => { await operate('system/admission/reject', schemas.decision, { version: 1, request_id: nextRequestID(), enrollment_request_id: request.request_id, reason }) })
                      }}><ShieldX aria-hidden="true" size={12} />拒绝</Button>
                    </div>}
                  </article>
                ))}
              </div>
            </section>

            <section className="admission-block" aria-labelledby="permits-title">
              <div className="admission-block-heading"><div><h3 id="permits-title">Permit</h3><p>{permits.length} 条集中记录；列表不返回 Permit 正文。</p></div></div>
              <div className="admission-records">
                {permits.length === 0 && <p className="admission-state">暂无 Permit。</p>}
                {permits.map((permit) => (
                  <article key={permit.permit_id}>
                    <div><strong>{shortID(permit.device_public_key_fingerprint)}</strong><span className={`admission-badge ${permit.status}`}>{permit.status}</span></div>
                    <p>目标 Node {permit.target_node_id}{permit.allow_descendants ? ' 及其后代' : ''} · {permit.admission_profile} · 到期 {dateTime(permit.expires_at_unix_ms)}</p>
                    {permit.status === 'active' && <div className="record-actions"><Button variant="secondary" size="sm" disabled={acting} onClick={() => {
                      const reason = window.prompt('撤销原因')?.trim()
                      if (reason) void act(async () => { await operate('system/admission/revoke-permit', schemas.revokePermit, { version: 1, request_id: nextRequestID(), permit_id: permit.permit_id, reason }) })
                    }}>撤销</Button></div>}
                  </article>
                ))}
              </div>
            </section>

            <section className="admission-block" aria-labelledby="enrollments-title">
              <div className="admission-block-heading"><div><h3 id="enrollments-title">已注册节点</h3><p>{enrollments.length} 条分配记录；已撤销 Node ID 保留 tombstone。</p></div></div>
              <div className="admission-records">
                {enrollments.length === 0 && <p className="admission-state">暂无已注册节点。</p>}
                {enrollments.map((enrollment) => (
                  <article key={enrollment.enrollment_id}>
                    <div><strong>Node {enrollment.node_id}</strong><span className={`admission-badge ${enrollment.status}`}>{enrollment.status}</span></div>
                    <p>父 Node {enrollment.parent_node_id} · {enrollment.admission_profile} · {shortID(enrollment.device_public_key_fingerprint)}</p>
                    {enrollment.reason && <p>{enrollment.reason}</p>}
                    {enrollment.status === 'active' && <div className="record-actions"><Button variant="secondary" size="sm" disabled={acting} onClick={() => {
                      const reason = window.prompt('撤销注册原因')?.trim()
                      if (reason) void act(async () => { await operate('system/admission/revoke-enrollment', schemas.revokeEnrollment, { version: 1, request_id: nextRequestID(), enrollment_id: enrollment.enrollment_id, reason }) })
                    }}>撤销注册</Button></div>}
                  </article>
                ))}
              </div>
            </section>
          </>
        )}
    </div>
  )
}
