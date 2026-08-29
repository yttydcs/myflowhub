import { Input } from './ui/input'
import type { Profile } from '../types'

export function createEmptyProfile(): Profile {
  return {
    id: '',
    name: '',
    node_id: '',
    endpoint: '',
    parent_node_id: '',
    parent_public_key: '',
    auto_connect: true,
  }
}

export function ProfileEditor({ profile, onChange, lockID = false }: {
  profile: Profile
  onChange(profile: Profile): void
  lockID?: boolean
}) {
  const update = (field: keyof Profile, value: string | boolean) => onChange({ ...profile, [field]: value })
  return (
    <div className="profile-fields">
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
        连接端点
        <Input required name="endpoint" autoComplete="off" spellCheck={false} value={profile.endpoint} onChange={(event) => update('endpoint', event.target.value)} placeholder="例如：127.0.0.1:7331" />
      </label>
      <label>
        父节点公钥
        <Input required name="parent-public-key" autoComplete="off" spellCheck={false} value={profile.parent_public_key} onChange={(event) => update('parent_public_key', event.target.value)} placeholder="Raw Base64 Ed25519 公钥" />
      </label>
      <label className="check-row">
        <input name="auto-connect" type="checkbox" checked={profile.auto_connect} onChange={(event) => update('auto_connect', event.target.checked)} />
        应用启动后自动连接此 Profile
      </label>
    </div>
  )
}
