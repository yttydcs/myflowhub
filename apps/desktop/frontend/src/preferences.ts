import { DEFAULT_EXPLORER_SPLIT_RATIO, isExplorerSplitRatio } from './lib/explorer-split'

export type Theme = 'light' | 'dark'
export type ExplorerCollapsedPane = 'node' | 'resource'

export interface UIPreferences {
  version: 1
  theme: Theme
  expanded_node_ids?: string[]
  expanded_resource_paths?: string[]
  focused_node_id?: string
  explorer_split_ratio?: number
  collapsed_explorer_pane?: ExplorerCollapsedPane
}

export interface LoadedUIPreferences {
  value: UIPreferences
  warning?: string
}

const STORAGE_PREFIX = 'mfh.desktop.ui.v1:'
const MAX_TREE_STATE_SIZE = 10_000

export function defaultUIPreferences(): UIPreferences {
  return { version: 1, theme: 'light', explorer_split_ratio: DEFAULT_EXPLORER_SPLIT_RATIO }
}

export function loadUIPreferences(profileID: string): LoadedUIPreferences {
  if (!profileID) return { value: defaultUIPreferences() }
  try {
    const raw = window.localStorage.getItem(storageKey(profileID))
    if (!raw) return { value: defaultUIPreferences() }
    const parsed = JSON.parse(raw) as unknown
    if (!isUIPreferences(parsed)) throw new Error('invalid preference document')
    return { value: { ...defaultUIPreferences(), ...parsed } }
  } catch {
    return {
      value: defaultUIPreferences(),
      warning: '界面偏好无法读取，已恢复浅色主题和默认浏览状态。',
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
  if (candidate.explorer_split_ratio !== undefined && !isExplorerSplitRatio(candidate.explorer_split_ratio)) return false
  if (candidate.collapsed_explorer_pane !== undefined
    && candidate.collapsed_explorer_pane !== 'node'
    && candidate.collapsed_explorer_pane !== 'resource') return false
  if (candidate.expanded_node_ids !== undefined && !isTreeState(candidate.expanded_node_ids)) return false
  if (candidate.expanded_resource_paths !== undefined && !isTreeState(candidate.expanded_resource_paths)) return false
  return true
}

function isTreeState(value: unknown): value is string[] {
  return Array.isArray(value)
    && value.length <= MAX_TREE_STATE_SIZE
    && value.every(isTreeID)
}

function isTreeID(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= 512
}
