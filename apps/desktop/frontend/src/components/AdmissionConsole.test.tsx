import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DesktopAPI } from '../api'
import { AdmissionConsole } from './AdmissionConsole'

function list(items: unknown[]) {
  return { schema: 'list', payload: { authority_epoch: 7, items } }
}

describe('AdmissionConsole', () => {
  afterEach(() => vi.restoreAllMocks())

  it('loads the centralized view and approves a pending request at the Authority owner', async () => {
    const requestID = '11'.repeat(16)
    const operate = vi.fn().mockImplementation(async (_owner: string, name: string) => {
      if (name === 'system/admission/list-permits') return list([])
      if (name === 'system/admission/list-requests') return list([{
        request_id: requestID,
        device_public_key_fingerprint: 'aa'.repeat(32),
        parent_node_id: '9',
        created_at_unix_ms: Date.now(),
        status: 'pending',
      }])
      if (name === 'system/admission/list-enrollments') return list([])
      return { schema: 'result', payload: { ok: true } }
    })

    render(<AdmissionConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" />)
    expect(await screen.findByText('父 Node 9', { exact: false })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '批准' }))

    await waitFor(() => expect(operate).toHaveBeenCalledWith(
      '1',
      'system/admission/approve',
      'invoke',
      'mfh.admission.decision.v1',
      expect.objectContaining({ enrollment_request_id: requestID, admission_profile: 'member' }),
    ))
  })

  it('issues a device-bound Permit without a client-selected Node ID', async () => {
    const permit = { version: 1, permit_id: '22'.repeat(16), signature: 'signed' }
    const operate = vi.fn().mockImplementation(async (_owner: string, name: string) => {
      if (name.startsWith('system/admission/list-')) return list([])
      if (name === 'system/admission/issue') return { schema: 'permit', payload: permit }
      return { schema: 'result', payload: { ok: true } }
    })
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })

    render(<AdmissionConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" />)
    await screen.findByText('暂无 Permit。')
    fireEvent.change(screen.getByLabelText('设备公钥或 SHA-256 指纹'), { target: { value: 'ab'.repeat(32) } })
    fireEvent.change(screen.getByLabelText('目标父 Node ID'), { target: { value: '9' } })
    fireEvent.click(screen.getByRole('button', { name: '签发 Permit' }))

    await waitFor(() => expect(operate).toHaveBeenCalledWith(
      '1',
      'system/admission/issue',
      'invoke',
      'mfh.admission.issue-permit.v1',
      expect.objectContaining({ device_public_key_fingerprint: 'ab'.repeat(32), target_node_id: '9' }),
    ))
    expect(screen.getByText(JSON.stringify(permit))).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '复制' }))
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(JSON.stringify(permit)))
  })

  it('shows an actionable error when the Authority view is forbidden or unavailable', async () => {
    const operate = vi.fn().mockRejectedValue(new Error('forbidden: management.admission.read'))
    render(<AdmissionConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('forbidden: management.admission.read')
  })

  it('follows stable cursors until every centralized record is loaded', async () => {
    const firstID = '31'.repeat(16)
    const secondID = '32'.repeat(16)
    const permit = (permitID: string, targetNodeID: string) => ({
      permit_id: permitID,
      device_public_key_fingerprint: 'ac'.repeat(32),
      target_node_id: targetNodeID,
      admission_profile: 'member',
      expires_at_unix_ms: Date.now() + 60_000,
      status: 'active',
    })
    const operate = vi.fn().mockImplementation(async (_owner: string, name: string, _capability: string, _schema: string, input: { cursor?: string }) => {
      if (name === 'system/admission/list-permits' && !input.cursor) {
        return { schema: 'list', payload: { authority_epoch: 7, items: [permit(firstID, '4')], next_cursor: firstID } }
      }
      if (name === 'system/admission/list-permits') return list([permit(secondID, '5')])
      return list([])
    })

    render(<AdmissionConsole api={{ operate } as unknown as DesktopAPI} authorityNodeID="1" />)
    expect(await screen.findByText('目标 Node 4', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('目标 Node 5', { exact: false })).toBeInTheDocument()
    expect(operate).toHaveBeenCalledWith(
      '1',
      'system/admission/list-permits',
      'invoke',
      'mfh.admission.list.v1',
      expect.objectContaining({ cursor: firstID }),
    )
  })
})
