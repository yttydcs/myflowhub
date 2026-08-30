import { describe, expect, it } from 'vitest'
import type { ResourceDescriptor } from '../types'
import {
  buildResourceTree,
  defaultExpandedResourcePaths,
  flattenResourceRows,
  resourcePathKey,
} from './resource-tree'

function resource(ownerNodeID: string, name: string, label?: string): ResourceDescriptor {
  return {
    id: { owner_node_id: ownerNodeID, name },
    type: 'mfh.variable',
    type_version: 1,
    capabilities: [],
    limits: { max_payload_bytes: 1024 },
    presentation: label ? { label } : undefined,
  }
}

describe('Resource path tree', () => {
  it('builds arbitrary-depth namespace, leaf, and hybrid rows', () => {
    const config = resource('2', 'system/config')
    const update = resource('2', 'system/config/update')
    const deep = resource('2', 'system/region/zone/rack/device/temperature')
    const index = buildResourceTree([deep, update, config])
    const systemKey = resourcePathKey('2', 'system')
    const configKey = resourcePathKey('2', 'system/config')

    expect(index.roots).toEqual([systemKey])
    expect(index.nodesByKey.get(systemKey)).toMatchObject({ segment: 'system' })
    expect(index.nodesByKey.get(systemKey)?.resource).toBeUndefined()
    expect(index.nodesByKey.get(configKey)).toMatchObject({ segment: 'config', resource: config })
    expect(index.nodesByKey.get(configKey)?.children).toEqual([resourcePathKey('2', 'system/config/update')])

    const rows = flattenResourceRows(index, new Set(index.nodesByKey.keys()))
    expect(rows.at(-1)).toMatchObject({ depth: 6, node: { segment: 'temperature', resource: deep } })
    expect(rows.find((row) => row.key === configKey)).toMatchObject({ depth: 2, node: { resource: config } })
  })

  it('keeps owner identities isolated and sorts segments numerically', () => {
    const index = buildResourceTree([
      resource('10', 'rack/port-10'),
      resource('2', 'rack/port-2'),
      resource('2', 'rack/port-10'),
    ])
    expect(index.roots).toEqual([resourcePathKey('2', 'rack'), resourcePathKey('10', 'rack')])
    expect(index.nodesByKey.get(resourcePathKey('2', 'rack'))?.children).toEqual([
      resourcePathKey('2', 'rack/port-2'),
      resourcePathKey('2', 'rack/port-10'),
    ])
  })

  it('defaults only first-level branches open and search temporarily reveals ancestors', () => {
    const index = buildResourceTree([
      resource('2', 'system/config/update', 'Apply configuration'),
      resource('2', 'metrics/cpu'),
    ])
    expect(defaultExpandedResourcePaths(index)).toEqual([
      resourcePathKey('2', 'metrics'),
      resourcePathKey('2', 'system'),
    ])
    expect(flattenResourceRows(index, new Set(), '').map((row) => row.node.path)).toEqual(['metrics', 'system'])
    expect(flattenResourceRows(index, new Set(), 'configuration').map((row) => row.node.path)).toEqual([
      'system',
      'system/config',
      'system/config/update',
    ])
  })

  it('rejects invalid or duplicate Resource identities explicitly', () => {
    expect(() => buildResourceTree([resource('2', 'system//health')])).toThrow('invalid Resource path')
    expect(() => buildResourceTree([resource('2', 'system/health'), resource('2', 'system/health')])).toThrow('duplicate Resource identity')
  })

  it('indexes and filters a 10k catalog within the desktop budget', () => {
    const resources = Array.from({ length: 10_000 }, (_, index) => resource('2', `metrics/group-${index % 100}/value-${index}`))
    const started = performance.now()
    const tree = buildResourceTree(resources)
    const rows = flattenResourceRows(tree, new Set(), 'value-9999')
    const elapsed = performance.now() - started
    expect(rows.map((row) => row.node.path)).toEqual(['metrics', 'metrics/group-99', 'metrics/group-99/value-9999'])
    expect(elapsed).toBeLessThan(750)
  })
})
