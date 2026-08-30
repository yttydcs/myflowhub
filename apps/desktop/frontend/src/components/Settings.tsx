import { useEffect, useState, type FormEvent } from 'react'
import { Cable, CircleUserRound, Moon, Palette, PlugZap, Plus, Save, ShieldCheck, Sun, Trash2, Unplug } from 'lucide-react'
import type { DesktopAPI } from '../api'
import { ProfileEditor, createEmptyProfile } from './ProfileEditor'
import { AdmissionConsole } from './AdmissionConsole'
import { Button } from './ui/button'
import type { Theme } from '../preferences'
import type { ConnectionStatus, Profile, Settings as SettingsDocument } from '../types'

type Section = 'connection' | 'admission' | 'profiles' | 'appearance'

export function Settings({
  settings,
  api,
  activeProfile,
  status,
  theme,
  busy,
  onConnect,
  onDisconnect,
  onSwitchProfile,
  onSaveProfile,
  onDeleteProfile,
  onThemeChange,
}: {
  settings: SettingsDocument
  api: DesktopAPI
  activeProfile: Profile
  status: ConnectionStatus
  theme: Theme
  busy: boolean
  onConnect(): Promise<void>
  onDisconnect(): Promise<void>
  onSwitchProfile(profileID: string): Promise<void>
  onSaveProfile(profile: Profile): Promise<boolean>
  onDeleteProfile(profileID: string): Promise<void>
  onThemeChange(theme: Theme): void
}) {
  const [section, setSection] = useState<Section>('connection')
  const [draft, setDraft] = useState<Profile>({ ...activeProfile })
  const [isNew, setIsNew] = useState(false)

  useEffect(() => {
    if (!isNew) setDraft({ ...activeProfile })
  }, [activeProfile, isNew])

  function editProfile(profile: Profile) {
    setDraft({ ...profile })
    setIsNew(false)
  }

  async function submitProfile(event: FormEvent) {
    event.preventDefault()
    const saved = await onSaveProfile(draft)
    if (saved) setIsNew(false)
  }

  async function deleteProfile(profile: Profile) {
    if (!window.confirm(`删除 Profile “${profile.name}”及其本地身份和视图？此操作不可恢复。`)) return
    await onDeleteProfile(profile.id)
  }

  return (
    <section className="settings-page" aria-labelledby="settings-title">
      <aside className="settings-nav" aria-label="设置分类">
        <h1 id="settings-title">设置</h1>
        <button className={section === 'connection' ? 'is-active' : ''} onClick={() => setSection('connection')}><Cable aria-hidden="true" size={15} />连接</button>
        {activeProfile.authority_node_id && <button className={section === 'admission' ? 'is-active' : ''} onClick={() => setSection('admission')}><ShieldCheck aria-hidden="true" size={15} />准入管理</button>}
        <button className={section === 'profiles' ? 'is-active' : ''} onClick={() => setSection('profiles')}><CircleUserRound aria-hidden="true" size={15} />Profile</button>
        <button className={section === 'appearance' ? 'is-active' : ''} onClick={() => setSection('appearance')}><Palette aria-hidden="true" size={15} />外观</button>
      </aside>

      <div className="settings-content">
        {section === 'connection' && (
          <div className="settings-section">
            <header><h2>连接</h2></header>
            <dl className="settings-facts">
              <div><dt>状态</dt><dd><span className={`status-dot ${status.state}`} aria-hidden="true" />{connectionLabel(status.state)}</dd></div>
              <div><dt>活动 Profile</dt><dd>{activeProfile.name}</dd></div>
              <div><dt>端点</dt><dd><code>{status.endpoint || activeProfile.endpoint}</code></dd></div>
              <div><dt>本机 Node</dt><dd>{activeProfile.node_id || '等待 Authority 下发'}</dd></div>
              <div><dt>父 Node</dt><dd>{status.parent_node_id || activeProfile.parent_node_id || '首次连接时确认'}</dd></div>
              <div><dt>凭据存储</dt><dd>{settings.credential_mode}</dd></div>
            </dl>
            {status.last_error && <p className="settings-error" role="alert">{status.last_error}</p>}
            <div className="settings-actions">
              {status.state === 'connected'
                ? <Button variant="secondary" disabled={busy} onClick={() => void onDisconnect()}><Unplug aria-hidden="true" size={14} />断开连接</Button>
                : <Button disabled={busy || status.state === 'connecting'} onClick={() => void onConnect()}><PlugZap aria-hidden="true" size={14} />{status.state === 'connecting' ? '连接中…' : '连接'}</Button>}
            </div>
          </div>
        )}

        {section === 'profiles' && (
          <div className="settings-section profile-settings">
            <header><h2>Profile</h2><Button variant="secondary" size="sm" onClick={() => { setDraft(createEmptyProfile()); setIsNew(true) }}><Plus aria-hidden="true" size={14} />新建 Profile</Button></header>
            <div className="profile-settings-body">
              <div className="settings-profile-list" aria-label="Profile 列表">
                {settings.profiles.map((profile) => (
                  <div key={profile.id} className={`settings-profile-row ${draft.id === profile.id && !isNew ? 'is-selected' : ''}`}>
                    <button onClick={() => editProfile(profile)}>
                      <span className="profile-avatar" aria-hidden="true">{profile.name.slice(0, 1).toUpperCase()}</span>
                      <span><strong>{profile.name}</strong><small>{profile.node_id || '未注册'} · {profile.endpoint}</small></span>
                    </button>
                    {profile.id !== activeProfile.id && <button className="activate-profile" onClick={() => void onSwitchProfile(profile.id)}>启用</button>}
                    <button className="profile-delete" onClick={() => void deleteProfile(profile)} aria-label={`删除 Profile ${profile.name}`}><Trash2 aria-hidden="true" size={13} /></button>
                  </div>
                ))}
              </div>
              <form className="settings-form" onSubmit={(event) => void submitProfile(event)} aria-busy={busy}>
                <h3>{isNew ? '新建 Profile' : `编辑 ${draft.name}`}</h3>
                <ProfileEditor profile={draft} onChange={setDraft} lockID={!isNew} />
                <div className="settings-actions"><Button type="submit" disabled={busy}><Save aria-hidden="true" size={14} />保存 Profile</Button></div>
              </form>
            </div>
          </div>
        )}

        {section === 'admission' && activeProfile.authority_node_id && <AdmissionConsole api={api} authorityNodeID={activeProfile.authority_node_id} />}

        {section === 'appearance' && (
          <div className="settings-section">
            <header><h2>外观</h2></header>
            <div className="appearance-setting">
              <div><strong>主题</strong><span>此选择仅应用于当前 Profile。</span></div>
              <div className="theme-options" role="radiogroup" aria-label="界面主题">
                <button role="radio" aria-checked={theme === 'light'} className={theme === 'light' ? 'is-selected' : ''} onClick={() => onThemeChange('light')}><Sun aria-hidden="true" size={15} />浅色</button>
                <button role="radio" aria-checked={theme === 'dark'} className={theme === 'dark' ? 'is-selected' : ''} onClick={() => onThemeChange('dark')}><Moon aria-hidden="true" size={15} />深色</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

function connectionLabel(state: ConnectionStatus['state']): string {
  return {
    signed_out: '未登录',
    disconnected: '已断开',
    connecting: '连接中',
    connected: '已连接',
    failed: '连接失败',
    stopped: '已停止',
  }[state]
}
