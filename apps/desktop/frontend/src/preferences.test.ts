import { afterEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_EXPLORER_SPLIT_RATIO } from './lib/explorer-split'
import { loadUIPreferences, saveUIPreferences } from './preferences'

describe('profile UI preferences', () => {
  afterEach(() => {
    window.localStorage.clear()
    vi.restoreAllMocks()
  })

  it('defaults to a light theme and keeps tree state isolated per Profile', () => {
    expect(loadUIPreferences('personal').value).toEqual({
      version: 1,
      theme: 'light',
      explorer_split_ratio: DEFAULT_EXPLORER_SPLIT_RATIO,
    })
    expect(saveUIPreferences('personal', {
      version: 1,
      theme: 'dark',
      expanded_node_ids: ['1', '2'],
      focused_node_id: '2',
      explorer_split_ratio: 0.62,
    })).toBeUndefined()
    expect(loadUIPreferences('personal').value).toMatchObject({ theme: 'dark', focused_node_id: '2', explorer_split_ratio: 0.62 })
    expect(loadUIPreferences('work').value.theme).toBe('light')
    expect(loadUIPreferences('work').value.explorer_split_ratio).toBe(DEFAULT_EXPLORER_SPLIT_RATIO)
  })

  it('loads older preference documents with the default split ratio', () => {
    window.localStorage.setItem('mfh.desktop.ui.v1:personal', JSON.stringify({ version: 1, theme: 'dark' }))
    expect(loadUIPreferences('personal').value).toEqual({
      version: 1,
      theme: 'dark',
      explorer_split_ratio: DEFAULT_EXPLORER_SPLIT_RATIO,
    })
  })

  it('rejects corrupted or oversized tree state with an explicit warning', () => {
    window.localStorage.setItem('mfh.desktop.ui.v1:personal', JSON.stringify({
      version: 1,
      theme: 'dark',
      expanded_node_ids: Array.from({ length: 10_001 }, (_, index) => String(index)),
    }))
    const loaded = loadUIPreferences('personal')
    expect(loaded.value).toEqual({
      version: 1,
      theme: 'light',
      explorer_split_ratio: DEFAULT_EXPLORER_SPLIT_RATIO,
    })
    expect(loaded.warning).toContain('无法读取')
  })

  it('rejects non-finite or out-of-range split ratios', () => {
    window.localStorage.setItem('mfh.desktop.ui.v1:personal', JSON.stringify({
      version: 1,
      theme: 'dark',
      explorer_split_ratio: 0.95,
    }))
    expect(loadUIPreferences('personal').warning).toContain('无法读取')
  })

  it('reports storage write failures instead of silently losing preferences', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota') })
    expect(saveUIPreferences('personal', { version: 1, theme: 'dark' })).toContain('保存失败')
  })
})
