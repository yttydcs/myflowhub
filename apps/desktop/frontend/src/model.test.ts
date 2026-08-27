import { describe, expect, it } from 'vitest'
import { errorText, pages, pretty } from './model'

describe('desktop presentation model', () => {
  it('keeps every canonical product surface addressable', () => {
    expect(pages.map((page) => page.id)).toEqual(expect.arrayContaining(['resources', 'metrics', 'clipboard', 'files', 'flows', 'authority']))
  })

  it('formats empty and error states without throwing', () => {
    expect(pretty({ nodes: [] })).toContain('nodes')
    expect(errorText(new Error('offline'))).toBe('offline')
  })
})
