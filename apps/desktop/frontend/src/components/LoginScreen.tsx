import { useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { ArrowRight, Check, Clock3, Copy, Moon, Pencil, Plus, ShieldCheck, Sun, Trash2 } from 'lucide-react'
import { BrandMark } from './BrandMark'
import { ProfileEditor, createEmptyProfile } from './ProfileEditor'
import { Button } from './ui/button'
import { Input, Textarea } from './ui/input'
import type { PreparedProfile, PublicIdentity } from '../api'
import type { Theme } from '../preferences'
import { generateProfileID } from '../profile-id'
import type { Profile, ProfileState, Settings } from '../types'

type Props = {
  settings: Settings
  profileStates: ProfileState[]
  busy: boolean
  error?: string
  theme: Theme
  onThemeChange(theme: Theme): void
  onPrepare(profile: Profile): Promise<PreparedProfile | undefined>
  onLogin(profile: Profile, permit: string, allowTOFU: boolean): Promise<boolean>
  onDelete(profileID: string): Promise<void>
}

type Entry = 'existing' | 'first'
type AdmissionMethod = 'permit' | 'approval'

function newAuthorityProfile(existing: Profile[]): Profile {
  return { ...createEmptyProfile(), id: generateProfileID(existing.map((profile) => profile.id)) }
}

export function LoginScreen({ settings, profileStates, busy, error, theme, onThemeChange, onPrepare, onLogin, onDelete }: Props) {
  const known = settings.profiles
  const stateByProfile = useMemo(() => new Map(profileStates.map((state) => [state.profile_id, state])), [profileStates])
  const [entry, setEntry] = useState<Entry>(() => known.length > 0 ? 'existing' : 'first')
  const [selectedID, setSelectedID] = useState(() => known[0]?.id || '')
  const [profile, setProfile] = useState<Profile>(() => newAuthorityProfile(known))
  const [permit, setPermit] = useState('')
  const [identity, setIdentity] = useState<PublicIdentity>()
  const [allowTOFU, setAllowTOFU] = useState(false)
  const [admissionMethod, setAdmissionMethod] = useState<AdmissionMethod>('approval')
  const [validationError, setValidationError] = useState('')
  const [copyStatus, setCopyStatus] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const errorRef = useRef<HTMLParagraphElement>(null)
  const selectedProfile = known.find((item) => item.id === selectedID) || known[0]
  const selectedState = selectedProfile ? stateByProfile.get(selectedProfile.id) : undefined
  const currentState = stateByProfile.get(profile.id)
  const enrollmentMode = profile.enrollment_mode || 'legacy'
  const isAuthority = enrollmentMode === 'authority'
  const isEnrolled = isAuthority && currentState?.state === 'enrolled'
  const isPending = isAuthority && currentState?.state === 'pending'
  const isSavedProfile = known.some((item) => item.id === profile.id)
  const needsTrustConfirmation = isAuthority
    && !isEnrolled
    && !isPending
    && admissionMethod === 'approval'
    && !profile.parent_public_key
    && !profile.authority_public_key
  const displayedError = validationError || error

  useEffect(() => { if (displayedError) errorRef.current?.focus() }, [displayedError])
  useEffect(() => {
    if (selectedID && known.some((item) => item.id === selectedID)) return
    setSelectedID(known[0]?.id || '')
  }, [known, selectedID])

  function resetTransient() {
    setPermit('')
    setIdentity(undefined)
    setAllowTOFU(false)
    setAdmissionMethod('approval')
    setValidationError('')
    setCopyStatus('')
  }

  function openFirst(next?: Profile) {
    setProfile(next ? { ...next } : newAuthorityProfile(known))
    resetTransient()
    setEntry('first')
  }

  function changeProfile(next: Profile) {
    setProfile(next)
    setIdentity(undefined)
    setAllowTOFU(false)
    setValidationError('')
    setCopyStatus('')
  }

  async function useSelectedProfile() {
    if (!selectedProfile || selectedState?.state === 'error') return
    if (!selectedState || selectedState.state === 'missing' || selectedState.state === 'device') {
      openFirst(selectedProfile)
      return
    }
    await onLogin(selectedProfile, '', false)
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (isAuthority && !isEnrolled && !isPending && admissionMethod === 'permit' && !permit.trim()) {
      setValidationError('请粘贴管理员签发的 Enrollment Permit。')
      return
    }
    if (needsTrustConfirmation && !allowTOFU) {
      setValidationError('请确认首次连接时信任该端点返回的父节点与 Authority 公钥。')
      return
    }
    setValidationError('')
    const selectedPermit = isAuthority && (isEnrolled || isPending || admissionMethod === 'approval') ? '' : permit
    if (await onLogin(profile, selectedPermit, needsTrustConfirmation ? allowTOFU : false)) setPermit('')
  }

  async function prepareIdentity() {
    if (!formRef.current?.reportValidity()) return
    const prepared = await onPrepare(profile)
    if (!prepared) return
    setProfile({ ...prepared.profile })
    setIdentity(prepared.identity)
    setCopyStatus('')
  }

  async function copyPublicKey() {
    if (!identity) return
    try {
      await navigator.clipboard.writeText(identity.public_key)
      setCopyStatus('公钥已复制')
    } catch {
      setCopyStatus('复制失败，请手动选择公钥')
    }
  }

  async function remove(item: Profile) {
    if (!window.confirm(`删除 Profile “${item.name}”及其本地身份和视图？此操作不可恢复。`)) return
    await onDelete(item.id)
    if (profile.id === item.id) openFirst()
  }

  function onTabKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
    event.preventDefault()
    const next = entry === 'existing' ? 'first' : 'existing'
    setEntry(next)
    event.currentTarget.parentElement?.querySelector<HTMLButtonElement>(`[data-entry="${next}"]`)?.focus()
  }

  const submitLabel = busy
    ? '正在连接…'
    : enrollmentMode === 'legacy'
      ? '兼容连接'
      : isEnrolled
        ? '连接父节点'
        : isPending
          ? '检查审批并连接'
          : admissionMethod === 'permit'
            ? '使用 Permit 注册并连接'
            : '提交注册申请'

  return (
    <main className="login-shell">
      <header className="login-topbar">
        <div className="brand-lockup"><BrandMark size="compact" /><strong>MyFlowHub</strong></div>
        <button className="icon-button" onClick={() => onThemeChange(theme === 'light' ? 'dark' : 'light')} aria-label={theme === 'light' ? '切换到深色主题' : '切换到浅色主题'}>
          {theme === 'light' ? <Moon aria-hidden="true" size={15} /> : <Sun aria-hidden="true" size={15} />}
        </button>
      </header>
      <section className="login-panel" aria-labelledby="login-title">
        <div className="login-heading"><BrandMark /><h1 id="login-title">登录 MyFlowHub</h1></div>
        <div className="login-entry-tabs" role="tablist" aria-label="登录方式">
          <button id="profile-existing-tab" data-entry="existing" role="tab" aria-controls="profile-existing-panel" aria-selected={entry === 'existing'} tabIndex={entry === 'existing' ? 0 : -1} className={entry === 'existing' ? 'is-active' : ''} onClick={() => setEntry('existing')} onKeyDown={onTabKeyDown}>使用现有 Profile</button>
          <button id="profile-first-tab" data-entry="first" role="tab" aria-controls="profile-first-panel" aria-selected={entry === 'first'} tabIndex={entry === 'first' ? 0 : -1} className={entry === 'first' ? 'is-active' : ''} onClick={() => setEntry('first')} onKeyDown={onTabKeyDown}>首次连接</button>
        </div>

        {entry === 'existing' ? (
          <div id="profile-existing-panel" className="existing-profile-entry" role="tabpanel" aria-labelledby="profile-existing-tab">
            {known.length === 0 ? (
              <div className="empty-profile-entry"><p>还没有保存的 Profile。</p><Button onClick={() => openFirst()}><Plus aria-hidden="true" size={14} />首次连接</Button></div>
            ) : (
              <>
                <div className="profile-choice-list" aria-label="已保存的 Profile">
                  {known.map((item) => {
                    const state = stateByProfile.get(item.id)
                    return (
                      <div key={item.id} className={`profile-choice ${selectedProfile?.id === item.id ? 'is-selected' : ''}`}>
                        <button className="profile-choice-main" onClick={() => setSelectedID(item.id)} disabled={busy} aria-pressed={selectedProfile?.id === item.id}>
                          <span className="profile-avatar" aria-hidden="true">{item.name.slice(0, 1).toUpperCase()}</span>
                          <span><strong>{item.name}</strong><small>{profileStateLabel(state, item)} · {item.endpoint}</small></span>
                        </button>
                        <button className="profile-row-action" onClick={() => openFirst(item)} disabled={busy} aria-label={`编辑 Profile ${item.name}`}><Pencil aria-hidden="true" size={13} /></button>
                        <button className="profile-row-action danger" onClick={() => void remove(item)} disabled={busy} aria-label={`删除 Profile ${item.name}`}><Trash2 aria-hidden="true" size={13} /></button>
                      </div>
                    )
                  })}
                </div>
                {selectedProfile && (
                  <div className="profile-choice-action">
                    <div><strong>{selectedProfile.name}</strong><span>{selectedState?.state === 'error' ? selectedState.message : profileStateHint(selectedState)}</span></div>
                    <Button disabled={busy || selectedState?.state === 'error'} onClick={() => void useSelectedProfile()}>{profileActionLabel(selectedState)}<ArrowRight aria-hidden="true" size={15} /></Button>
                  </div>
                )}
              </>
            )}
            {displayedError && <p ref={errorRef} className="form-error" role="alert" aria-live="polite" tabIndex={-1}>{displayedError}</p>}
          </div>
        ) : (
          <form id="profile-first-panel" ref={formRef} className="login-form first-connection-form" role="tabpanel" aria-labelledby="profile-first-tab" aria-busy={busy} onSubmit={(event) => void submit(event)}>
            <ProfileEditor profile={profile} onChange={changeProfile} lockID={isSavedProfile} hideID layout="login" />

            {isAuthority && !isEnrolled && !isPending && (
              <fieldset className="admission-methods compact-admission-methods">
                <legend>准入方式</legend>
                <label className={admissionMethod === 'approval' ? 'is-selected' : ''}>
                  <input type="radio" name="admission-method" checked={admissionMethod === 'approval'} onChange={() => { setAdmissionMethod('approval'); setAllowTOFU(false); setValidationError('') }} />
                  <Clock3 aria-hidden="true" size={16} /><span><strong>申请管理员审批</strong></span>
                </label>
                <label className={admissionMethod === 'permit' ? 'is-selected' : ''}>
                  <input type="radio" name="admission-method" checked={admissionMethod === 'permit'} onChange={() => { setAdmissionMethod('permit'); setAllowTOFU(false); setValidationError('') }} />
                  <ShieldCheck aria-hidden="true" size={16} /><span><strong>已有 Permit</strong></span>
                </label>
              </fieldset>
            )}

            {isPending && <div className="pending-profile-note"><Clock3 aria-hidden="true" size={16} /><span><strong>正在等待管理员审批</strong>{currentState?.request_id && <small>Request {currentState.request_id}</small>}</span></div>}

            {isAuthority && !isEnrolled && !isPending && admissionMethod === 'permit' && (
              <div className="permit-entry">
                <div className="identity-action"><span>{identity ? '本机公钥已准备' : '签发 Permit 前需要本机公钥'}</span><Button type="button" variant="secondary" size="sm" disabled={busy} onClick={() => void prepareIdentity()}>{identity ? '重新读取' : isSavedProfile ? '读取本机公钥' : '生成本机公钥'}</Button></div>
                {identity && (
                  <div className="identity-result">
                    <label>本机公钥<span className="public-key-row"><Input readOnly spellCheck={false} value={identity.public_key} onFocus={(event) => event.currentTarget.select()} /><Button type="button" variant="secondary" size="sm" onClick={() => void copyPublicKey()} aria-label="复制本机公钥">{copyStatus === '公钥已复制' ? <Check aria-hidden="true" size={13} /> : <Copy aria-hidden="true" size={13} />}复制</Button></span></label>
                    {copyStatus && <p className="copy-status" aria-live="polite">{copyStatus}</p>}
                  </div>
                )}
                <label className="permit-field">Enrollment Permit<Textarea name="admission-permit" autoComplete="off" spellCheck={false} aria-invalid={validationError.includes('Enrollment Permit') || undefined} value={permit} onChange={(event) => { setPermit(event.target.value); setValidationError('') }} rows={3} placeholder={'粘贴一次性 Permit，例如：{"version":1,"permit_id":"…"}'} /></label>
              </div>
            )}

            {isAuthority && needsTrustConfirmation && (
              <label className="check-row trust-confirmation"><input name="allow-tofu" type="checkbox" checked={allowTOFU} onChange={(event) => { setAllowTOFU(event.target.checked); setValidationError('') }} />首次连接时信任该端点返回的父节点与 Authority 公钥</label>
            )}

            {enrollmentMode === 'legacy' && (
              <label className="permit-field">Legacy Join Permit <span className="optional">（可选）</span><Textarea name="admission-permit" autoComplete="off" spellCheck={false} value={permit} onChange={(event) => setPermit(event.target.value)} rows={3} /></label>
            )}

            {displayedError && <p ref={errorRef} className="form-error" role="alert" aria-live="polite" tabIndex={-1}>{displayedError}</p>}
            <div className="login-submit"><Button type="submit" disabled={busy}>{submitLabel}<ArrowRight aria-hidden="true" size={15} /></Button></div>
          </form>
        )}
      </section>
    </main>
  )
}

function profileStateLabel(state: ProfileState | undefined, profile: Profile): string {
  if (!state) return '正在读取状态'
  if (state.state === 'legacy') return `Legacy · Node ${profile.node_id}`
  if (state.state === 'enrolled') return `Node ${state.node_id || profile.node_id}`
  return ({ missing: '尚未准备', device: '设备已准备', pending: '等待审批', error: '凭据异常' })[state.state]
}

function profileStateHint(state: ProfileState | undefined): string {
  if (!state) return '正在读取本机凭据状态。'
  return ({ legacy: '使用旧版身份兼容连接。', missing: '继续首次连接以创建设备身份。', device: '设备身份已保存，可继续审批或 Permit 注册。', pending: '复用原申请检查审批结果，不会重复确认信任。', enrolled: '注册已完成，可直接连接父节点。', error: state.message || '受保护凭据不可用。' })[state.state]
}

function profileActionLabel(state: ProfileState | undefined): string {
  if (!state) return '继续'
  return ({ legacy: '兼容连接', missing: '继续首次连接', device: '继续首次连接', pending: '检查审批并连接', enrolled: '连接父节点', error: '需要恢复' })[state.state]
}
