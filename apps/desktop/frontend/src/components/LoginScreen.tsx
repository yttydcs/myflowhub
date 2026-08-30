import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ArrowRight, BadgeCheck, Check, Clock3, Copy, Fingerprint, Moon, Network, Plus, ShieldCheck, Sun, Trash2 } from 'lucide-react'
import { BrandMark } from './BrandMark'
import { ProfileEditor, createEmptyProfile } from './ProfileEditor'
import { Button } from './ui/button'
import { Input, Textarea } from './ui/input'
import type { PreparedProfile, PublicIdentity } from '../api'
import type { Theme } from '../preferences'
import type { Profile, Settings } from '../types'

type Props = {
  settings: Settings
  busy: boolean
  error?: string
  theme: Theme
  onThemeChange(theme: Theme): void
  onPrepare(profile: Profile): Promise<PreparedProfile | undefined>
  onLogin(profile: Profile, permit: string, allowTOFU: boolean): Promise<boolean>
  onDelete(profileID: string): Promise<void>
}

type AdmissionMethod = 'permit' | 'approval'

export function LoginScreen({ settings, busy, error, theme, onThemeChange, onPrepare, onLogin, onDelete }: Props) {
  const known = useMemo(() => settings.profiles, [settings.profiles])
  const initial = useMemo(
    () => known.find((profile) => profile.id === settings.active_profile_id) || known[0] || createEmptyProfile(),
    [known, settings.active_profile_id],
  )
  const [profile, setProfile] = useState<Profile>({ ...initial })
  const [permit, setPermit] = useState('')
  const [identity, setIdentity] = useState<PublicIdentity>()
  const [allowTOFU, setAllowTOFU] = useState(false)
  const [admissionMethod, setAdmissionMethod] = useState<AdmissionMethod>('approval')
  const [validationError, setValidationError] = useState('')
  const [copyStatus, setCopyStatus] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const errorRef = useRef<HTMLParagraphElement>(null)
  const enrollmentMode = profile.enrollment_mode || 'legacy'
  const isAuthority = enrollmentMode === 'authority'
  const isEnrolled = isAuthority && Boolean(profile.node_id)
  const isSavedProfile = known.some((item) => item.id === profile.id)
  const needsTrustConfirmation = isAuthority
    && !isEnrolled
    && admissionMethod === 'approval'
    && !profile.parent_public_key
    && !profile.authority_public_key
  const displayedError = validationError || error

  useEffect(() => { if (displayedError) errorRef.current?.focus() }, [displayedError])

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (isAuthority && !isEnrolled && admissionMethod === 'permit' && !permit.trim()) {
      setValidationError('请粘贴管理员签发的 Enrollment Permit。')
      return
    }
    if (needsTrustConfirmation && !allowTOFU) {
      setValidationError('首次无 Permit 申请需要确认信任当前父节点端点，或在高级设置中预置公钥。')
      return
    }
    setValidationError('')
    const selectedPermit = isAuthority && (isEnrolled || admissionMethod === 'approval') ? '' : permit
    const selectedTOFU = needsTrustConfirmation ? allowTOFU : false
    if (await onLogin(profile, selectedPermit, selectedTOFU)) setPermit('')
  }

  function changeProfile(next: Profile) {
    setProfile(next)
    setIdentity(undefined)
    setAllowTOFU(false)
    setValidationError('')
    setCopyStatus('')
  }

  function selectProfile(next: Profile) {
    changeProfile(next)
    setPermit('')
    setAdmissionMethod('approval')
  }

  function selectAdmissionMethod(next: AdmissionMethod) {
    setAdmissionMethod(next)
    setAllowTOFU(false)
    setValidationError('')
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
    if (profile.id === item.id) selectProfile(createEmptyProfile())
  }

  const submitLabel = busy
    ? '正在连接…'
    : enrollmentMode === 'legacy'
      ? '登录并进入工作区'
      : isEnrolled
        ? '连接父节点'
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
        <div className="login-heading">
          <BrandMark />
          <h1 id="login-title">登录 MyFlowHub</h1>
        </div>

        {known.length > 0 && (
          <div className="known-profiles" aria-label="已保存的 Profile">
            <div className="section-label"><span>已保存的 Profile</span><button onClick={() => selectProfile(createEmptyProfile())}><Plus aria-hidden="true" size={13} />新建</button></div>
            {known.map((item) => (
              <div key={item.id} className={`profile-row ${profile.id === item.id ? 'is-selected' : ''}`}>
                <button className="profile-open" onClick={() => selectProfile({ ...item })} disabled={busy}>
                  <span className="profile-avatar" aria-hidden="true">{item.name.slice(0, 1).toUpperCase()}</span>
                  <span><strong>{item.name}</strong><small>{item.node_id ? `Node ${item.node_id}` : '等待注册'} · {item.endpoint}</small></span>
                </button>
                <button className="profile-delete" onClick={() => void remove(item)} disabled={busy} aria-label={`删除 Profile ${item.name}`}><Trash2 aria-hidden="true" size={13} /></button>
              </div>
            ))}
          </div>
        )}

        <form ref={formRef} className="login-form" aria-busy={busy} onSubmit={(event) => void submit(event)}>
          <section className="login-step" aria-labelledby="connection-step-title">
            <header className="login-step-heading">
              <span className="login-step-index" aria-hidden="true">01</span>
              <div>
                <h2 id="connection-step-title"><Network aria-hidden="true" size={15} />连接到父节点</h2>
                <p>填写本地 Profile 和父节点地址。常规注册无需输入 Node ID 或公钥。</p>
              </div>
            </header>
            <div className="login-step-body">
              <ProfileEditor profile={profile} onChange={changeProfile} lockID={isSavedProfile} layout="login" />
            </div>
          </section>

          <section className="login-step" aria-labelledby="identity-preparation-title">
            <header className="login-step-heading">
              <span className="login-step-index" aria-hidden="true">02</span>
              <div>
                <h2 id="identity-preparation-title"><Fingerprint aria-hidden="true" size={15} />{isEnrolled ? '确认设备身份' : '准备这台设备'}</h2>
                <p>{isEnrolled
                  ? 'Node ID 已由 Authority 分配；连接时会读取本地密钥和注册凭据。'
                  : isAuthority
                    ? '首次连接会自动创建设备密钥。若管理员要提前签发 Permit，可先生成并复制公钥。'
                    : '读取旧版身份对应的受保护设备密钥。'}</p>
              </div>
            </header>
            <div className="login-step-body">
              <div className={`identity-preparation ${identity ? 'is-ready' : ''} ${isEnrolled ? 'is-enrolled' : ''}`}>
                <div className="identity-action">
                  <div className="identity-state">
                    {isEnrolled ? <BadgeCheck aria-hidden="true" size={18} /> : <Fingerprint aria-hidden="true" size={18} />}
                    <span>
                      <strong>{isEnrolled ? `已注册为 Node ${profile.node_id}` : identity ? '设备身份已准备' : isSavedProfile ? '本机已有设备身份' : '首次连接时创建设备身份'}</strong>
                      <small>{isEnrolled ? '无需再次选择 Permit 或提交审批' : identity ? 'Node ID 将在准入成功后下发' : isSavedProfile ? '可读取公钥或继续原来的审批申请' : '私钥只保存在这台设备上'}</small>
                    </span>
                  </div>
                  <Button type="button" variant="secondary" size="sm" disabled={busy} onClick={() => void prepareIdentity()}>
                    {identity ? '重新读取' : isEnrolled || isSavedProfile ? '查看公钥' : '提前生成公钥'}
                  </Button>
                </div>
                {identity && (
                  <div className="identity-result">
                    <label>
                      本机公钥
                      <span className="public-key-row">
                        <Input readOnly spellCheck={false} value={identity.public_key} onFocus={(event) => event.currentTarget.select()} />
                        <Button type="button" variant="secondary" size="sm" onClick={() => void copyPublicKey()} aria-label="复制本机公钥">
                          {copyStatus === '公钥已复制' ? <Check aria-hidden="true" size={13} /> : <Copy aria-hidden="true" size={13} />}复制
                        </Button>
                      </span>
                    </label>
                    <p className="identity-note">{identity.node_id ? `Node ${identity.node_id}` : 'Node ID 尚未分配'} · Profile 已保存，但尚未登录或连接。</p>
                    {copyStatus && <p className="copy-status" aria-live="polite">{copyStatus}</p>}
                  </div>
                )}
              </div>
            </div>
          </section>

          {isAuthority && !isEnrolled && (
            <section className="login-step" aria-labelledby="admission-step-title">
              <header className="login-step-heading">
                <span className="login-step-index" aria-hidden="true">03</span>
                <div>
                  <h2 id="admission-step-title"><ShieldCheck aria-hidden="true" size={15} />选择准入方式</h2>
                  <p>有 Permit 就直接注册；没有 Permit 则提交一次申请，由管理员审批。</p>
                </div>
              </header>
              <div className="login-step-body">
                <fieldset className="admission-methods">
                  <legend className="sr-only">准入方式</legend>
                  <label className={admissionMethod === 'permit' ? 'is-selected' : ''}>
                    <input type="radio" name="admission-method" value="permit" checked={admissionMethod === 'permit'} onChange={() => selectAdmissionMethod('permit')} />
                    <ShieldCheck aria-hidden="true" size={16} />
                    <span><strong>已有 Permit</strong><small>使用管理员提前签发的一次性许可，直接注册并连接</small></span>
                  </label>
                  <label className={admissionMethod === 'approval' ? 'is-selected' : ''}>
                    <input type="radio" name="admission-method" value="approval" checked={admissionMethod === 'approval'} onChange={() => selectAdmissionMethod('approval')} />
                    <Clock3 aria-hidden="true" size={16} />
                    <span><strong>申请管理员审批</strong><small>先提交设备申请；批准后再次连接即可完成注册</small></span>
                  </label>
                </fieldset>

                {admissionMethod === 'permit'
                  ? (
                    <label className="permit-field">
                      Enrollment Permit
                      <Textarea
                        name="admission-permit"
                        autoComplete="off"
                        spellCheck={false}
                        aria-invalid={validationError.includes('Enrollment Permit') || undefined}
                        value={permit}
                        onChange={(event) => { setPermit(event.target.value); setValidationError('') }}
                        rows={3}
                        placeholder={'粘贴一次性 Permit，例如：{"version":1,"permit_id":"…"}'}
                      />
                    </label>
                  )
                  : (
                    <div className="approval-path">
                      <p>提交后会生成稳定的审批请求，不会分配临时 Node ID。管理员批准后，请使用同一 Profile 再次连接。</p>
                      {needsTrustConfirmation && (
                        <label className="check-row trust-confirmation">
                          <input name="allow-tofu" type="checkbox" checked={allowTOFU} onChange={(event) => { setAllowTOFU(event.target.checked); setValidationError('') }} />
                          我确认首次连接时信任该端点返回的父节点与 Authority 公钥；注册成功后将固定校验
                        </label>
                      )}
                    </div>
                  )}
              </div>
            </section>
          )}

          {enrollmentMode === 'legacy' && (
            <section className="login-step" aria-labelledby="legacy-permit-title">
              <header className="login-step-heading">
                <span className="login-step-index" aria-hidden="true">03</span>
                <div>
                  <h2 id="legacy-permit-title"><ShieldCheck aria-hidden="true" size={15} />旧版准入许可</h2>
                  <p>仅在父节点要求旧版 Join Permit 时填写；已经准入的身份可留空。</p>
                </div>
              </header>
              <div className="login-step-body">
                <label className="permit-field">
                  Legacy Join Permit <span className="optional">（可选）</span>
                  <Textarea name="admission-permit" autoComplete="off" spellCheck={false} value={permit} onChange={(event) => setPermit(event.target.value)} rows={3} placeholder={'例如：{"version":1,"permit_id":"…"}'} />
                </label>
              </div>
            </section>
          )}

          {displayedError && <p ref={errorRef} className="form-error" role="alert" aria-live="polite" tabIndex={-1}>{displayedError}</p>}
          <div className="login-submit">
            <p>{isEnrolled
              ? '设备身份已注册，本次只建立到父节点的连接。'
              : admissionMethod === 'permit' || enrollmentMode === 'legacy'
                ? '连接成功后会保存此 Profile。'
                : '审批期间可关闭应用，之后使用同一 Profile 重试。'}</p>
            <Button type="submit" disabled={busy}>{submitLabel} <ArrowRight aria-hidden="true" size={15} /></Button>
          </div>
        </form>
      </section>
    </main>
  )
}
