import { describe, expect, it } from 'vitest'
import { generateProfileID } from './profile-id'

describe('Profile ID generation', () => {
  it('uses valid stable-width lowercase IDs and retries collisions', () => {
    let call = 0
    const random = () => new Uint8Array(8).fill(call++ === 0 ? 0x11 : 0x22)
    const id = generateProfileID(['profile-1111111111111111'], random)
    expect(id).toBe('profile-2222222222222222')
    expect(id).toMatch(/^[a-z0-9][a-z0-9._-]{0,63}$/)
  })

  it('fails explicitly after bounded collisions', () => {
    const random = () => new Uint8Array(8).fill(0x33)
    expect(() => generateProfileID(['profile-3333333333333333'], random)).toThrow('无法生成唯一')
  })

  it('rejects a malformed random source', () => {
    expect(() => generateProfileID([], () => new Uint8Array(7))).toThrow('错误长度')
  })
})
