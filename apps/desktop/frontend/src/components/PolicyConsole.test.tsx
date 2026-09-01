import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import { PolicyConsole } from './PolicyConsole'

const definition = {
  version: 1,
  id: 'superadmin',
  label: 'Super administrator',
  revision: 1,
  immutable: true,
  rules: [{ resource: { kind: 'all' }, capability: { kind: 'all' } }],
}

function emptyPage(members: { key: string }[] = []) {
  return { schema: 'mfh.collection.page.v1', payload: { version: 1, revision: 7, members } }
}

describe('PolicyConsole', () => {
  afterEach(() => vi.restoreAllMocks())

  it('loads Authority policy collections and creates an explicit Subject binding', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const operate = vi.fn().mockImplementation(async (_owner: string, name: string, capability: string) => {
      if (capability === 'list') {
        return name === 'system/policy/definitions' ? emptyPage([{ key: 'superadmin' }]) : emptyPage()
      }
      if (capability === 'get' && name === 'system/policy/definitions') return { schema: 'definition', payload: definition }
      return { schema: 'result', payload: { version: 1, status: 'ok' } }
    })
    render(<PolicyConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" currentNodeID="41" />)

		expect((await screen.findAllByText('Super administrator')).length).toBeGreaterThan(0)
		expect(screen.getByLabelText('Subject Node ID')).toHaveValue('41')
    fireEvent.click(screen.getByRole('button', { name: '创建 Binding' }))

    await waitFor(() => expect(operate).toHaveBeenCalledWith(
      '1',
      'system/policy/bindings',
      'create',
      'mfh.policy.binding-create.v1',
      expect.objectContaining({
        subject: '41',
        definition_id: 'superadmin',
        scope: { kind: 'authority-domain', node_id: '1' },
      }),
    ))
  })

  it('shows Authority Forbidden as the final policy result', async () => {
    const operate = vi.fn().mockRejectedValue(new Error('forbidden: Authority-domain superadmin binding is required'))
    render(<PolicyConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" currentNodeID="41" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Authority-domain superadmin binding is required')
  })
})
