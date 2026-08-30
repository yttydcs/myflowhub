// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { MetricsAPI } from './api'
import type { Configuration, Definition, Status } from './contracts'
import { createMetricsApp } from './app'

const definitions: Definition[] = [
  {name: 'cpu_percent', unit: 'percent', controllable: false, interval_ms: 2000},
  {name: 'volume_percent', unit: 'percent', controllable: true, interval_ms: 1000},
]

const configuration: Configuration = {
  version: 1,
  revision: 4,
  platform: 'windows',
  settings: [
    {metric: 'cpu_percent', enabled: true, writable: false, interval_ms: 2000},
    {metric: 'volume_percent', enabled: true, writable: true, interval_ms: 1000},
  ],
  notification_channels: ['system'],
}

function statusValue(running = true): Status {
  return {
    version: 1,
    running,
    ...(running ? {
      connection: {state: 'connected', parent_node_id: '1', endpoint: '127.0.0.1:7341', attempt: 1, link_generation: 7},
      configuration,
      samples: [
        {version: 1, metric: 'cpu_percent', value: '18', unit: 'percent', status: 'fresh', sampled_at_unix_ms: 1000, observed_at_unix_ms: 1001},
        {version: 1, metric: 'volume_percent', value: '34', unit: 'percent', status: 'stale', sampled_at_unix_ms: 900, observed_at_unix_ms: 1001, error: 'collector timeout'},
      ],
    } : {samples: []}),
    notifications: {queue_depth: 0, dropped: 0},
  }
}

function fakeAPI(initialStatus = statusValue()): MetricsAPI {
  return {
    identity: vi.fn(async () => ({node_id: '20', public_key: 'public-key'})),
    start: vi.fn(async () => statusValue()),
    stop: vi.fn(async () => undefined),
    status: vi.fn(async () => initialStatus),
    configuration: vi.fn(async () => configuration),
    definitions: vi.fn(async () => definitions),
    updateConfiguration: vi.fn(async value => ({...value, revision: value.revision + 1})),
  }
}

function mount(api: MetricsAPI) {
  const root = document.createElement('main')
  root.id = 'app'
  document.body.replaceChildren(root)
  const app = createMetricsApp(root, api, {storage: window.localStorage, now: () => 1_725_000_000_000})
  return {app, root}
}

beforeEach(() => {
  window.localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
})

afterEach(() => {
  vi.useRealTimers()
  document.body.replaceChildren()
})

describe('Metrics application', () => {
  it('renders only backend samples and supports navigation, selection and theme persistence', async () => {
    const api = fakeAPI()
    const {app, root} = mount(api)
    await app.ready

    expect(root.textContent).toContain('处理器')
    expect(root.textContent).toContain('18%')
    expect(root.textContent).toContain('collector timeout')
    expect(root.querySelectorAll('[data-metric]')).toHaveLength(2)
    expect(root.querySelector('[data-metric="cpu_percent"]')?.getAttribute('aria-pressed')).toBe('true')

    root.querySelector<HTMLButtonElement>('[data-metric="volume_percent"]')!.click()
    expect(root.querySelector('[data-metric="volume_percent"]')?.getAttribute('aria-pressed')).toBe('true')
    expect(root.querySelector('#inspector-content')?.textContent).toContain('系统音量')

    root.querySelector<HTMLButtonElement>('[data-view="policy"]')!.click()
    expect(root.querySelector<HTMLElement>('[data-page="policy"]')!.hidden).toBe(false)
    expect(root.querySelector<HTMLInputElement>('[data-policy-metric="cpu_percent"] [data-field="writable"]')!.disabled).toBe(true)

    root.querySelector<HTMLButtonElement>('#theme-button')!.click()
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(window.localStorage.getItem('myflowhub.metrics.theme.v1')).toBe('dark')
    app.dispose()
  })

  it('validates a permit, guards duplicate start, and clears the secret after success', async () => {
    const api = fakeAPI(statusValue(false))
    const {app, root} = mount(api)
    await app.ready
    root.querySelector<HTMLInputElement>('#parent-key')!.value = 'raw-public-key'
    root.querySelector<HTMLTextAreaElement>('#permit')!.value = '{"version":1}'

    const start = root.querySelector<HTMLButtonElement>('#start-button')!
    start.click()
    start.click()
    await vi.waitFor(() => expect(vi.mocked(api.start)).toHaveBeenCalledTimes(1))
    expect(vi.mocked(api.start).mock.calls[0][0]).toMatchObject({node_id: '20', permit: {version: 1}})
    expect(root.querySelector<HTMLTextAreaElement>('#permit')!.value).toBe('')
    expect(JSON.stringify(window.localStorage)).not.toContain('permit')
    app.dispose()
  })

  it('keeps invalid permit errors visible without calling the backend', async () => {
    const api = fakeAPI(statusValue(false))
    const {app, root} = mount(api)
    await app.ready
    root.querySelector<HTMLInputElement>('#parent-key')!.value = 'raw-public-key'
    root.querySelector<HTMLTextAreaElement>('#permit')!.value = '{'
    root.querySelector<HTMLButtonElement>('#start-button')!.click()

    await vi.waitFor(() => expect(root.querySelector('#error-banner')?.textContent).toContain('有效 JSON 对象'))
    expect(api.start).not.toHaveBeenCalled()
    app.dispose()
  })

  it('saves normalized policy input using the current revision', async () => {
    const api = fakeAPI()
    const {app, root} = mount(api)
    await app.ready
    root.querySelector<HTMLButtonElement>('[data-view="policy"]')!.click()
    root.querySelector<HTMLInputElement>('#notification-channels')!.value = 'zeta, system, zeta'
    root.querySelector<HTMLInputElement>('[data-policy-metric="cpu_percent"] [data-field="interval"]')!.value = '2500'
    root.querySelector<HTMLButtonElement>('#save-policy-button')!.click()

    await vi.waitFor(() => expect(api.updateConfiguration).toHaveBeenCalledTimes(1))
    expect(vi.mocked(api.updateConfiguration).mock.calls[0][0]).toMatchObject({
      revision: 4,
      notification_channels: ['system', 'zeta'],
      settings: expect.arrayContaining([{metric: 'cpu_percent', enabled: true, writable: false, interval_ms: 2500}]),
    })
    await vi.waitFor(() => expect(root.querySelector('#policy-page-revision')?.textContent).toBe('revision 5'))
    app.dispose()
  })

  it('stops scheduling status polls after dispose', async () => {
    vi.useFakeTimers()
    const api = fakeAPI()
    const {app} = mount(api)
    await app.ready
    expect(api.status).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1000)
    expect(api.status).toHaveBeenCalledTimes(2)
    app.dispose()
    await vi.advanceTimersByTimeAsync(5000)
    expect(api.status).toHaveBeenCalledTimes(2)
  })
})
