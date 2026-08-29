import { afterEach, describe, expect, it, vi } from 'vitest'
import { loadUIPreferences, saveUIPreferences } from './preferences'

describe('profile UI preferences', () => {
  afterEach(() => {
    window.localStorage.clear()
    vi.restoreAllMocks()
  })

  it('defaults to a light theme and keeps tree state isolated per Profile', () => {
    expect(loadUIPreferences('personal').value).toEqual({ version: 1, theme: 'light' })
    expect(saveUIPreferences('personal', {
      version: 1,
      theme: 'dark',
      expanded_node_ids: ['1', '2'],
      focused_node_id: '2',
    })).toBeUndefined()
    expect(loadUIPreferences('personal').value).toMatchObject({ theme: 'dark', focused_node_id: '2' })
    expect(loadUIPreferences('work').value.theme).toBe('light')
  })

  it('rejects corrupted or oversized tree state with an explicit warning', () => {
    window.localStorage.setItem('mfh.desktop.ui.v1:personal', JSON.stringify({
      version: 1,
      theme: 'dark',
      expanded_node_ids: Array.from({ length: 10_001 }, (_, index) => String(index)),
    }))
    const loaded = loadUIPreferences('personal')
    expect(loaded.value).toEqual({ version: 1, theme: 'light' })
    expect(loaded.warning).toContain('无法读取')
  })

  it('reports storage write failures instead of silently losing preferences', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota') })
    expect(saveUIPreferences('personal', { version: 1, theme: 'dark' })).toContain('保存失败')
  })
})
