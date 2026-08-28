import { useMemo, useState, type FormEvent } from 'react'
import { ArrowRight, KeyRound, Network, Pencil, ShieldCheck, Trash2 } from 'lucide-react'
import { Button } from './ui/button'
import { Input, Textarea } from './ui/input'
import type { Profile, Settings } from '../types'

type Props = {
  settings: Settings
  busy: boolean
  error?: string
  onLogin(profile: Profile, permit: string): Promise<void>
  onSwitch(profileID: string): Promise<void>
  onDelete(profileID: string): Promise<void>
}

const emptyProfile: Profile = {
  id: '',
  name: '',
  node_id: '',
  endpoint: '',
  parent_node_id: '',
  parent_public_key: '',
  auto_connect: true,
}

export function LoginScreen({ settings, busy, error, onLogin, onSwitch, onDelete }: Props) {
  const [profile, setProfile] = useState<Profile>(emptyProfile)
  const [permit, setPermit] = useState('')
  const known = useMemo(() => settings.profiles, [settings.profiles])

  const update = (field: keyof Profile, value: string | boolean) => setProfile((current) => ({ ...current, [field]: value }))
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    await onLogin(profile, permit)
    setPermit('')
  }
  const remove = async (item: Profile) => {
    if (!window.confirm(`删除 Profile “${item.name}”及其本地身份和视图？此操作不可恢复。`)) return
    await onDelete(item.id)
    if (profile.id === item.id) setProfile(emptyProfile)
  }

  return (
    <main className="login-shell">
      <section className="login-intro" aria-labelledby="login-title">
        <div className="brand-mark" aria-hidden="true">M</div>
        <p className="eyebrow">MYFLOWHUB / RESOURCE WORKSPACE</p>
        <h1 id="login-title">让设备成为节点，<br />让能力成为资源。</h1>
        <p className="login-copy">连接一棵受控的节点树，在同一个安静的工作区里发现、订阅和组合资源。</p>
        <div className="login-principles">
          <span><Network size={16} /> 链路可替换</span>
          <span><ShieldCheck size={16} /> 父节点裁决</span>
          <span><KeyRound size={16} /> 凭据按 Profile 隔离</span>
        </div>
      </section>

      <section className="login-panel" aria-label="登录到节点树">
        <div className="panel-heading">
          <div><p className="eyebrow">新建连接</p><h2>登录 Profile</h2></div>
          <span className="quiet-label">permit 仅用于本次登录</span>
        </div>
        {known.length > 0 && (
          <div className="known-profiles" aria-label="已有 Profile">
            {known.map((item) => (
              <div key={item.id} className="profile-row">
                <button className="profile-open" onClick={() => void onSwitch(item.id)} disabled={busy}>
                  <span className="profile-avatar">{item.name.slice(0, 1).toUpperCase()}</span>
                  <span><strong>{item.name}</strong><small>{item.node_id} · {item.endpoint}</small></span>
                  <ArrowRight size={16} />
                </button>
                <div className="profile-actions">
                  <button onClick={() => setProfile({ ...item })} disabled={busy} aria-label={`编辑 Profile ${item.name}`}><Pencil size={14} /></button>
                  <button onClick={() => void remove(item)} disabled={busy} aria-label={`删除 Profile ${item.name}`}><Trash2 size={14} /></button>
                </div>
              </div>
            ))}
          </div>
        )}
        <form className="login-form" onSubmit={(event) => void submit(event)}>
          <label>Profile 名称<Input required value={profile.name} onChange={(event) => update('name', event.target.value)} placeholder="个人工作区" /></label>
          <label>Profile ID<Input required pattern="[a-z0-9][a-z0-9._-]{0,63}" value={profile.id} onChange={(event) => update('id', event.target.value)} placeholder="personal" /></label>
          <div className="form-pair">
            <label>本机 Node ID<Input required inputMode="numeric" value={profile.node_id} onChange={(event) => update('node_id', event.target.value)} placeholder="2" /></label>
            <label>父 Node ID<Input required inputMode="numeric" value={profile.parent_node_id} onChange={(event) => update('parent_node_id', event.target.value)} placeholder="1" /></label>
          </div>
          <label>连接端点<Input required value={profile.endpoint} onChange={(event) => update('endpoint', event.target.value)} placeholder="127.0.0.1:9540" /></label>
          <label>父节点公钥<Input required value={profile.parent_public_key} onChange={(event) => update('parent_public_key', event.target.value)} placeholder="Raw Base64 Ed25519 public key" /></label>
          <label>一次性准入 Permit <span className="optional">（已准入可留空）</span><Textarea value={permit} onChange={(event) => setPermit(event.target.value)} rows={3} placeholder="{ ... }" /></label>
          <label className="check-row"><input type="checkbox" checked={profile.auto_connect} onChange={(event) => update('auto_connect', event.target.checked)} /> 下次启动自动连接</label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <Button type="submit" disabled={busy}>{busy ? '正在建立受控链路…' : '登录并进入工作区'} <ArrowRight size={16} /></Button>
        </form>
      </section>
    </main>
  )
}
