import { describe, expect, it, vi } from 'vitest'
import { acquireVariable } from './controller'
import { VariableOverviewStore } from './overview-store'
import { overviewFixture } from './overview.test-support'

describe('independent overview subscription lifetime', () => {
  it('keeps 2000 subscriptions receiving while no row is mounted and shares the editor session', async () => {
    const { api, resources, push } = overviewFixture()
    const store = new VariableOverviewStore(api, 'profile')
    store.watch(resources)
    try {
      await vi.waitFor(() => expect(store.getSnapshot().live).toBe(2000), { timeout: 10_000 })
      expect(api.subscribe).toHaveBeenCalledTimes(2000)
      push('load/1999', true)
      await vi.waitFor(() => expect(store.controller(resources[1999]!)?.getSnapshot().snapshot?.value).toBe(true))
      const editor = acquireVariable(api, 'profile', resources[0]!)
      editor.controller.edit(true)
      expect(editor.controller).toBe(store.controller(resources[0]!))
      store.stop()
      expect(editor.controller.getSnapshot().dirty).toBe(true)
      expect(api.cancel).toHaveBeenCalledTimes(1999)
      editor.release()
      expect(api.cancel).toHaveBeenCalledTimes(2000)
    } finally { store.stop() }
  })
})
