import { Input } from './ui/input'
import type { Profile } from '../types'

export function createEmptyProfile(): Profile {
  return {
    id: '',
    name: '',
    enrollment_mode: 'authority',
    node_id: '',
    endpoint: '',
    parent_node_id: '',
    parent_public_key: '',
    auto_connect: true,
  }
}

export function ProfileEditor({ profile, onChange, lockID = false, layout = 'settings' }: {
  profile: Profile
  onChange(profile: Profile): void
  lockID?: boolean
  layout?: 'settings' | 'login'
}) {
  // Profiles created before enrollment support have no mode and remain legacy.
  const enrollmentMode = profile.enrollment_mode || 'legacy'
  const update = (field: keyof Profile, value: string | boolean) => onChange({ ...profile, [field]: value })

  const modeSelector = (
    <fieldset className="enrollment-mode">
      <legend>{layout === 'login' ? '连接协议' : '注册方式'}</legend>
      <label className={enrollmentMode === 'authority' ? 'is-selected' : ''}>
        <input type="radio" name="enrollment-mode" value="authority" checked={enrollmentMode === 'authority'} onChange={() => update('enrollment_mode', 'authority')} />
        <span><strong>Authority 自动注册</strong><small>Node ID 与信任信息由系统在首次准入后下发</small></span>
      </label>
      <label className={enrollmentMode === 'legacy' ? 'is-selected' : ''}>
        <input type="radio" name="enrollment-mode" value="legacy" checked={enrollmentMode === 'legacy'} onChange={() => update('enrollment_mode', 'legacy')} />
        <span><strong>Legacy 兼容</strong><small>仅用于已有 Node ID 和父节点公钥的旧版身份</small></span>
      </label>
    </fieldset>
  )

  const legacyFields = (
    <div className="mode-specific-fields">
      <div className="settings-field-grid">
        <label>
          本机 Node ID
          <Input required name="node-id" autoComplete="off" inputMode="numeric" pattern="[1-9][0-9]*" value={profile.node_id} onChange={(event) => update('node_id', event.target.value)} placeholder="例如：2" />
        </label>
        <label>
          父 Node ID
          <Input required name="parent-node-id" autoComplete="off" inputMode="numeric" pattern="[1-9][0-9]*" value={profile.parent_node_id} onChange={(event) => update('parent_node_id', event.target.value)} placeholder="例如：1" />
        </label>
      </div>
      <label>
        父节点公钥
        <Input required name="parent-public-key" autoComplete="off" spellCheck={false} value={profile.parent_public_key} onChange={(event) => update('parent_public_key', event.target.value)} placeholder="Raw Base64 Ed25519 公钥" />
      </label>
      <details className="advanced-enrollment nested-enrollment">
        <summary>集中准入管理</summary>
        <p>仅在此 Legacy 身份需要从非 Authority 父节点管理集中准入时填写。</p>
        <label>
          Admission Authority Node ID <span className="optional">（可选）</span>
          <Input name="authority-node-id" autoComplete="off" inputMode="numeric" pattern="[1-9][0-9]*" value={profile.authority_node_id || ''} onChange={(event) => update('authority_node_id', event.target.value)} placeholder="例如：1" />
        </label>
      </details>
    </div>
  )

  const authorityTrustFields = (
    <div className="mode-specific-fields trust-pin-fields">
      <p>通常无需填写。只有预先掌握身份锚点时才固定这些值。</p>
      <div className="settings-field-grid">
        <label>
          预期父 Node ID <span className="optional">（可选）</span>
          <Input name="parent-node-id" autoComplete="off" inputMode="numeric" pattern="[1-9][0-9]*" value={profile.parent_node_id} onChange={(event) => update('parent_node_id', event.target.value)} placeholder="首次连接时获取" />
        </label>
        <label>
          父节点公钥 <span className="optional">（可选）</span>
          <Input name="parent-public-key" autoComplete="off" spellCheck={false} value={profile.parent_public_key} onChange={(event) => update('parent_public_key', event.target.value)} placeholder="Raw Base64 Ed25519 公钥" />
        </label>
      </div>
      <label>
        Authority 公钥 <span className="optional">（可选）</span>
        <Input name="authority-public-key" autoComplete="off" spellCheck={false} value={profile.authority_public_key || ''} onChange={(event) => update('authority_public_key', event.target.value)} placeholder="Raw Base64 Ed25519 公钥" />
      </label>
      {profile.authority_node_id && <p>已绑定 Authority Node {profile.authority_node_id}</p>}
    </div>
  )

  const modeFields = enrollmentMode === 'legacy' ? legacyFields : authorityTrustFields

  return (
    <div className={`profile-fields ${layout === 'login' ? 'login-profile-fields' : ''}`}>
      <div className="settings-field-grid">
        <label>
          Profile 名称
          <Input required name="profile-name" autoComplete="off" value={profile.name} onChange={(event) => update('name', event.target.value)} placeholder="例如：个人工作区" />
        </label>
        <label>
          Profile ID
          <Input
            required
            name="profile-id"
            autoComplete="off"
            spellCheck={false}
            pattern="[a-z0-9][a-z0-9._-]{0,63}"
            value={profile.id}
            onChange={(event) => update('id', event.target.value)}
            placeholder="例如：personal"
            readOnly={lockID}
          />
        </label>
      </div>
      {layout === 'settings' && modeSelector}
      <label>
        父节点地址
        <Input required name="endpoint" autoComplete="off" spellCheck={false} value={profile.endpoint} onChange={(event) => update('endpoint', event.target.value)} placeholder="例如：127.0.0.1:7331" />
      </label>
      {layout === 'settings' && enrollmentMode === 'authority' && profile.node_id && (
        <p className="assigned-node">这台设备已注册为 Node <strong>{profile.node_id}</strong></p>
      )}
      {layout === 'login'
        ? (
          <details className="advanced-enrollment login-advanced-enrollment">
            <summary>高级与兼容连接设置</summary>
            <div className="login-advanced-body">
              {modeSelector}
              {modeFields}
            </div>
          </details>
        )
        : enrollmentMode === 'legacy'
          ? legacyFields
          : (
            <details className="advanced-enrollment">
              <summary>高级信任设置</summary>
              {authorityTrustFields}
            </details>
          )}
      <label className="check-row">
        <input name="auto-connect" type="checkbox" checked={profile.auto_connect} onChange={(event) => update('auto_connect', event.target.checked)} />
        应用启动后自动连接此 Profile
      </label>
    </div>
  )
}
