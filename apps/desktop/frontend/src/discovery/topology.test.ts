import { describe, expect, it } from 'vitest'
import { node, query } from './fixtures.test-support'
import { parseTopology, topologyCoverage } from './topology'

describe('topology response validation', () => {
  it('accepts a bounded partial tree and only covers complete child lists', () => {
    const response = query('1', [node('1', undefined, true), node('2', '1', true)])
    expect(parseTopology(response, '1', 1)).toEqual(response)
    expect([...topologyCoverage(response)]).toEqual([['1', ['2']]])
  })

  it.each([
    { version: 2 }, { depth: 0 }, { revision: 0 }, { revision: Number.MAX_SAFE_INTEGER + 1 },
    { revision: 1.5 }, { instance_id: 'A'.repeat(32) }, { instance_id: 'abc' }, { extra: true }, { nodes: [] },
  ])('rejects invalid envelope %j', (patch) => {
    expect(() => parseTopology({ ...query('1', [node('1')]), ...patch }, '1', 1)).toThrow()
  })

  it.each([
    [node('1'), node('1')],
    [node('2')],
    [node('1', '2')],
    [node('1'), node('2', '9')],
    [node('1'), node('2', '3', true), node('3', '2', true)],
    [node('1', undefined, true), node('2', '1', true), node('3', '2')],
    [node('1', undefined, true)],
    [node('1'), node('2', '1')],
    [{ ...node('1'), has_children: undefined }],
    [{ ...node('1'), has_children: 0 }],
    [{ ...node('1'), generation: 0 }],
    [{ ...node('1'), parent_id: null }],
    [{ ...node('1'), display_name: '字'.repeat(86) }],
    [{ ...node('1'), role: '' }],
    [{ ...node('1'), unexpected: 1 }],
  ])('rejects malformed or incomplete membership %j', (...nodes) => {
    expect(() => parseTopology(query('1', nodes as ReturnType<typeof node>[]), '1', 1)).toThrow()
  })

  it('validates a 4096-level tree iteratively and rejects overflow', () => {
    const nodes = Array.from({ length: 4096 }, (_, index) => node(String(index + 1), index ? String(index) : undefined, index < 4095))
    expect(parseTopology(query('1', nodes, 0), '1', 0).nodes).toHaveLength(4096)
    expect(() => parseTopology(query('1', [...nodes, node('4097', '4096')], 0), '1', 0)).toThrow('4096')
  })

  it('enforces the payload limit as well as the node count', () => {
    const nodes = [node('1', undefined, true), ...Array.from({ length: 3000 }, (_, i) => ({ ...node(String(i + 2), '1'), display_name: 'x'.repeat(255) }))]
    expect(() => parseTopology(query('1', nodes), '1', 1)).toThrow('MiB')
  })
})
