import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ArrowRight, Moon, Plus, Sun, Trash2 } from 'lucide-react'
import { ProfileEditor, createEmptyProfile } from './ProfileEditor'
import { Button } from './ui/button'
import { Textarea } from './ui/input'
import type { Theme } from '../preferences'
import type { Profile, Settings } from '../types'

type Props = {
  settings: Settings
  busy: boolean
  error?: string
  theme: Theme
  onThemeChange(theme: Theme): void
  onLogin(profile: Profile, permit: string): Promise<boolean>
  onDelete(profileID: string): Promise<void>
}

export function LoginScreen({ settings, busy, error, theme, onThemeChange, onLogin, onDelete }: Props) {
  const known = useMemo(() => settings.profiles, [settings.profiles])
  const initial = useMemo(
    () => known.find((profile) => profile.id === settings.active_profile_id) || known[0] || createEmptyProfile(),
    [known, settings.active_profile_id],
  )
  const [profile, setProfile] = useState<Profile>({ ...initial })
  const [permit, setPermit] = useState('')
  const errorRef = useRef<HTMLParagraphElement>(null)

  useEffect(() => { if (error) errorRef.current?.focus() }, [error])

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (await onLogin(profile, permit)) setPermit('')
  }

  async function remove(item: Profile) {
    if (!window.confirm(`删除 Profile “${item.name}”及其本地身份和视图？此操作不可恢复。`)) return
    await onDelete(item.id)
    if (profile.id === item.id) setProfile(createEmptyProfile())
  }

  return (
    <main className="login-shell">
      <header className="login-topbar">
        <div className="brand-lockup"><span className="brand-mark small" aria-hidden="true">M</span><strong>MyFlowHub</strong></div>
        <button className="icon-button" onClick={() => onThemeChange(theme === 'light' ? 'dark' : 'light')} aria-label={theme === 'light' ? '切换到深色主题' : '切换到浅色主题'}>
          {theme === 'light' ? <Moon aria-hidden="true" size={15} /> : <Sun aria-hidden="true" size={15} />}
        </button>
      </header>
      <section className="login-panel" aria-labelledby="login-title">
        <div className="login-heading">
          <span className="brand-mark" aria-hidden="true">M</span>
          <h1 id="login-title">登录 MyFlowHub</h1>
        </div>

        {known.length > 0 && (
          <div className="known-profiles" aria-label="已保存的 Profile">
            <div className="section-label"><span>已保存的 Profile</span><button onClick={() => { setProfile(createEmptyProfile()); setPermit('') }}><Plus aria-hidden="true" size={13} />新建</button></div>
            {known.map((item) => (
              <div key={item.id} className={`profile-row ${profile.id === item.id ? 'is-selected' : ''}`}>
                <button className="profile-open" onClick={() => { setProfile({ ...item }); setPermit('') }} disabled={busy}>
                  <span className="profile-avatar" aria-hidden="true">{item.name.slice(0, 1).toUpperCase()}</span>
                  <span><strong>{item.name}</strong><small>{item.node_id} · {item.endpoint}</small></span>
                </button>
                <button className="profile-delete" onClick={() => void remove(item)} disabled={busy} aria-label={`删除 Profile ${item.name}`}><Trash2 aria-hidden="true" size={13} /></button>
              </div>
            ))}
          </div>
        )}

        <form className="login-form" aria-busy={busy} onSubmit={(event) => void submit(event)}>
          <ProfileEditor profile={profile} onChange={setProfile} lockID={known.some((item) => item.id === profile.id)} />
          <label>
            一次性准入 Permit <span className="optional">（已准入可留空）</span>
            <Textarea name="admission-permit" autoComplete="off" spellCheck={false} value={permit} onChange={(event) => setPermit(event.target.value)} rows={3} placeholder={'例如：{"version":1,"permit_id":"…"}'} />
          </label>
          {error && <p ref={errorRef} className="form-error" role="alert" aria-live="polite" tabIndex={-1}>{error}</p>}
          <Button type="submit" disabled={busy}>{busy ? '正在连接…' : '登录并进入工作区'} <ArrowRight aria-hidden="true" size={15} /></Button>
        </form>
      </section>
    </main>
  )
}
