export type Theme = 'light' | 'dark'

export interface UIPreferences {
  version: 1
  theme: Theme
  expanded_node_ids?: string[]
  focused_node_id?: string
}

export interface LoadedUIPreferences {
  value: UIPreferences
  warning?: string
}

const STORAGE_PREFIX = 'mfh.desktop.ui.v1:'
const MAX_TREE_STATE_SIZE = 10_000

export function defaultUIPreferences(): UIPreferences {
  return { version: 1, theme: 'light' }
}

export function loadUIPreferences(profileID: string): LoadedUIPreferences {
  if (!profileID) return { value: defaultUIPreferences() }
  try {
    const raw = window.localStorage.getItem(storageKey(profileID))
    if (!raw) return { value: defaultUIPreferences() }
    const parsed = JSON.parse(raw) as unknown
    if (!isUIPreferences(parsed)) throw new Error('invalid preference document')
    return { value: parsed }
  } catch {
    return {
      value: defaultUIPreferences(),
      warning: '界面偏好无法读取，已恢复浅色主题和默认树状态。',
    }
  }
}

export function saveUIPreferences(profileID: string, preferences: UIPreferences): string | undefined {
  if (!profileID) return '尚未选择 Profile，无法保存界面偏好。'
  try {
    window.localStorage.setItem(storageKey(profileID), JSON.stringify(preferences))
    return undefined
  } catch {
    return '界面偏好保存失败。请检查应用存储权限或可用空间。'
  }
}

function storageKey(profileID: string): string {
  return `${STORAGE_PREFIX}${profileID}`
}

function isUIPreferences(value: unknown): value is UIPreferences {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<UIPreferences>
  if (candidate.version !== 1 || (candidate.theme !== 'light' && candidate.theme !== 'dark')) return false
  if (candidate.focused_node_id !== undefined && !isTreeID(candidate.focused_node_id)) return false
  if (candidate.expanded_node_ids === undefined) return true
  return Array.isArray(candidate.expanded_node_ids)
    && candidate.expanded_node_ids.length <= MAX_TREE_STATE_SIZE
    && candidate.expanded_node_ids.every(isTreeID)
}

function isTreeID(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= 512
}
