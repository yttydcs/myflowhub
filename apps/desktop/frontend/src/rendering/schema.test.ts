import { describe, expect, it } from 'vitest'
import type { ResourceDescriptor } from '../types'
import { defaultDataValue, resolveSchema, validateDataValue } from './schema'
import { rendererChoices, selectResourceRenderer, validateRendererSettings } from './registry'

const resource = (type: string, schema: string, writable = false): ResourceDescriptor => ({
  id: { owner_node_id: '1', name: 'example/value' },
  type,
  type_version: 1,
  capabilities: [
    { name: 'read', permission: 'read', output_schema: schema, max_payload_bytes: 1024 },
    ...(writable ? [{ name: 'write', permission: 'write', input_schema: 'mfh.variable-write.v2', max_payload_bytes: 1024 }] : []),
  ],
  schemas: [{ id: schema, content_type: 'application/json' }],
  limits: { max_payload_bytes: 1024 },
})

describe('schema resolver and renderer registry', () => {
  it('resolves generated first-party definitions', () => {
    const result = resolveSchema('mfh.management.health.v1')
    expect(result.status).toBe('resolved')
    if (result.status === 'resolved') expect(result.schema.properties?.some((property) => property.name === 'state')).toBe(true)
    expect(selectResourceRenderer(resource('mfh.variable', 'mfh.management.health.v1')).selected.id).toBe('mfh.structured.health.v1')
  })

  it('returns an actionable unsupported result', () => {
    expect(resolveSchema('example.scalar.v1')).toEqual(expect.objectContaining({ status: 'unsupported', schemaID: 'example.scalar.v1' }))
  })

  it('rejects unsupported provider keywords before rendering', () => {
    const result = resolveSchema('example.invalid.v1', [{
      id: 'test-provider',
      resolve: () => ({
        status: 'resolved',
        schemaID: 'example.invalid.v1',
        source: 'test-provider',
        schema: { type: 'string', oneOf: [] } as never,
      }),
    }])
    expect(result).toEqual(expect.objectContaining({ status: 'invalid', reason: expect.stringContaining('oneOf') }))
  })

  it('validates numeric bounds and step', () => {
    const schema = { type: 'integer' as const, minimum: 0, maximum: 100, multiple_of: 1 }
    expect(validateDataValue(schema, 42)).toEqual([])
    expect(validateDataValue(schema, 42.5).map((error) => error.message)).toContain('需要整数')
    expect(validateDataValue(schema, 101).map((error) => error.message)).toContain('不能大于 100')
  })

  it('validates provider string length and pattern constraints', () => {
    const schema = { type: 'string' as const, min_length: 3, max_length: 8, pattern: '^[a-z]+$' }
    expect(validateDataValue(schema, 'worker')).toEqual([])
    expect(validateDataValue(schema, 'A1').map((error) => error.message)).toEqual(expect.arrayContaining(['至少 3 个字符', '文本格式不符合要求']))
  })

  it('only offers a slider when provider bounds make it meaningful', () => {
    const bounded = rendererChoices(resource('mfh.variable', 'example.bounded.v1', true), {
      status: 'resolved',
      schemaID: 'example.bounded.v1',
      source: 'test',
      schema: { type: 'integer', minimum: 0, maximum: 100, multiple_of: 1 },
    })
    const unbounded = rendererChoices(resource('mfh.variable', 'example.unbounded.v1', true), {
      status: 'resolved',
      schemaID: 'example.unbounded.v1',
      source: 'test',
      schema: { type: 'integer', multiple_of: 1 },
    })
    expect(bounded.map((choice) => choice.id)).toContain('mfh.variable.slider.v1')
    expect(unbounded.map((choice) => choice.id)).not.toContain('mfh.variable.slider.v1')
  })

  it('builds required object defaults and validates missing fields', () => {
    const schema = { type: 'object' as const, properties: [{ name: 'version', required: true, schema: { type: 'integer' as const, minimum: 1 } }] }
    expect(defaultDataValue(schema)).toEqual({ version: 1 })
    expect(validateDataValue(schema, {})).toEqual([{ path: '$.version', message: '必填' }])
  })

  it('keeps unknown schemas on raw fallback', () => {
    const selection = selectResourceRenderer(resource('mfh.variable', 'example.scalar.v1'), 'mfh.variable.slider.v1')
    expect(selection.selected.id).toBe('mfh.variable.raw.v1')
    expect(selection.fallbackReason).toContain('不兼容')
  })

  it('rejects renderer settings outside the phase-1 empty allowlist', () => {
    expect(validateRendererSettings('mfh.variable.number.v1', undefined)).toEqual({})
    expect(validateRendererSettings('mfh.variable.number.v1', {})).toEqual({})
    expect(validateRendererSettings('mfh.variable.number.v1', { value: 42 }).reason).toContain('不接受')
  })

  it('offers generated form and JSON modes for known commands', () => {
    const command = resource('mfh.command', 'mfh.management.issue-permit.v1')
    command.capabilities = [{ name: 'invoke', permission: 'invoke', input_schema: 'mfh.management.issue-permit.v1', output_schema: 'mfh.provisioning.permit.v1', max_payload_bytes: 4096 }]
    const selection = selectResourceRenderer(command, 'mfh.command')
    expect(rendererChoices(command, selection.resolution).map((choice) => choice.id)).toEqual(['mfh.operation.form.v1', 'mfh.operation.json.v1'])
  })
})
