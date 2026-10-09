import { SchemaVariableValueV1, SchemaVariableWriteV3, type JSONValue, type DataSchemaDefinition, type VariableValueV1 } from '../generated/variable-contract.generated'
import type { ResourceDescriptor, ResourceEvent } from '../types'

export const VARIABLE_VALUE_SCHEMA = SchemaVariableValueV1
export const VARIABLE_WRITE_SCHEMA = SchemaVariableWriteV3
export const MAX_VARIABLE_BYTES = 1024 * 1024
const bytes = (text: string) => new TextEncoder().encode(text).byteLength
const encodedBytes = (value: unknown) => bytes(JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, (char) => `\\u${char.charCodeAt(0).toString(16).padStart(4, '0')}`))
const own = (object: object, key: string) => Object.prototype.hasOwnProperty.call(object, key)
function fail(message: string): never { throw new Error(message) }
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) fail('需要 JSON 对象')
  return value as Record<string, unknown>
}
function keys(value: Record<string, unknown>, allowed: string[]) {
  for (const key of Object.keys(value)) if (!allowed.includes(key)) fail(`unsupported 字段 ${key}`)
}
function text(value: unknown, limit: number, required = false) {
  if (typeof value !== 'string' || (required && !value) || bytes(value) > limit) fail('文本无效或超过字节上限')
  validateString({ type: 'string' }, value)
}
function hex(value: unknown) { if (typeof value !== 'string' || !/^[0-9a-f]{32}$/.test(value)) fail('instance_id 必须为 32 位小写十六进制') }
export function parseRevision(value: unknown): bigint {
  if (typeof value !== 'string' || !/^[1-9][0-9]{0,19}$/.test(value) || BigInt(value) > 18446744073709551615n) fail('revision 必须为规范 uint64 字符串')
  return BigInt(value)
}
export function variablePayloadLimit(resource: ResourceDescriptor, capability: string) {
  return Math.min(MAX_VARIABLE_BYTES, resource.limits.max_payload_bytes, resource.capabilities.find((item) => item.name === capability)?.max_payload_bytes ?? MAX_VARIABLE_BYTES)
}
export function descriptorError(resource: ResourceDescriptor): string | undefined {
  if (resource.type !== 'mfh.variable' || resource.type_version !== 2) return `unsupported Variable type_version ${resource.type_version}`
  for (const capability of resource.capabilities) {
    if (capability.name === 'read' && capability.output_schema !== VARIABLE_VALUE_SCHEMA) return 'unsupported Variable read schema'
    if (capability.name === 'subscribe' && capability.event_schema !== VARIABLE_VALUE_SCHEMA) return 'unsupported Variable subscribe schema'
    if (capability.name === 'write' && (capability.input_schema !== VARIABLE_WRITE_SCHEMA || capability.output_schema !== VARIABLE_VALUE_SCHEMA)) return 'unsupported Variable write schema'
  }
  if (!resource.capabilities.some((item) => item.name === 'read' || item.name === 'subscribe')) return 'Variable 没有 read/subscribe 能力'
  if (!Number.isSafeInteger(resource.limits.max_payload_bytes) || resource.limits.max_payload_bytes <= 0) return 'Variable payload limit 无效'
}

// Parse before materializing untrusted JSON: bound bytes/depth, reject duplicate
// keys, and keep decimal spellings for the same exact step tests as protocol.
export function parseBoundedJSON(raw: string, limit = MAX_VARIABLE_BYTES) {
  if (bytes(raw) > limit) fail('JSON 超过字节上限')
  let position = 0
  const numbers = new Map<string, string>()
  const space = () => { while (' \t\r\n'.includes(raw[position] || '\0')) position++ }
  function string(): string {
    const start = position++
    while (position < raw.length) {
      const char = raw[position++]
      if (char === '\\') position++
      else if (char === '"') {
        const result = JSON.parse(raw.slice(start, position)) as string
        validateString({ type: 'string' }, result)
        return result
      }
    }
    return fail('JSON string 未结束')
  }
  function read(path: string[], depth: number): unknown {
    if (depth > 48) fail('JSON 深度超限')
    space()
    const char = raw[position]
    if (char === '"') return string()
    if (char === '{' || char === '[') {
      position++
      const array = char === '['
      const output: unknown[] | Record<string, unknown> = array ? [] : Object.create(null)
      const seen = new Set<string>()
      space()
      if (raw[position] === (array ? ']' : '}')) { position++; return output }
      let index = 0
      while (position < raw.length) {
        space()
        let key = String(index++)
        if (!array) {
          if (raw[position] !== '"') fail('JSON 对象键无效')
          key = string(); space()
          if (seen.has(key)) fail('JSON 重复对象键')
          seen.add(key)
          if (raw[position++] !== ':') fail('JSON 缺少冒号')
        }
        const value = read([...path, key], depth + 1)
        if (array) (output as unknown[]).push(value)
        else (output as Record<string, unknown>)[key] = value
        space()
        const separator = raw[position++]
        if (separator === (array ? ']' : '}')) return output
        if (separator !== ',') fail('JSON 分隔符无效')
      }
      return fail('JSON 未结束')
    }
    for (const [literal, value] of [['true', true], ['false', false], ['null', null]] as const) {
      if (raw.startsWith(literal, position)) { position += literal.length; return value }
    }
    const match = /^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?/.exec(raw.slice(position))
    if (!match) return fail('无效 JSON value')
    position += match[0].length
    const number = Number(match[0])
    if (!Number.isFinite(number)) fail('JSON 数值必须有限')
    numbers.set(JSON.stringify(path), match[0])
    return number
  }
  const value = read([], 1)
  space()
  if (position !== raw.length) fail('JSON 存在尾随内容')
  return { value, numbers }
}

const schemaKeys = ['id', 'type', 'title', 'description', 'format', 'unit', 'precision', 'minimum', 'maximum', 'multiple_of', 'min_length', 'max_length', 'pattern', 'min_items', 'max_items', 'enum', 'read_only', 'write_only', 'sensitive', 'nullable', 'content_type', 'properties', 'items', 'additional_properties']
export function validateVariableDefinition(value: unknown): asserts value is DataSchemaDefinition {
  let properties = 0
  function visit(value: unknown, depth: number) {
    if (depth > 12) fail('definition 超过 12 层')
    const schema = record(value)
    keys(schema, schemaKeys)
    if (typeof schema.type !== 'string' || !['null', 'boolean', 'integer', 'number', 'string', 'object', 'array'].includes(schema.type)) fail('unsupported definition type')
    for (const key of ['id', 'title', 'description', 'format', 'unit', 'pattern', 'content_type']) if (own(schema, key)) text(schema[key], key === 'id' ? 255 : key === 'content_type' ? 127 : 1024)
    for (const key of ['nullable', 'read_only', 'write_only', 'sensitive']) if (own(schema, key) && typeof schema[key] !== 'boolean') fail(`${key} 需要 boolean`)
    if (schema.read_only && schema.write_only || schema.sensitive && !schema.write_only) fail('definition 读写注解无效')
    const numeric = schema.type === 'integer' || schema.type === 'number'
    for (const key of ['minimum', 'maximum', 'multiple_of', 'precision']) {
      if (!own(schema, key)) continue
      if (!numeric || typeof schema[key] !== 'number' || !Number.isFinite(schema[key]) || Math.abs(schema[key] as number) > Number.MAX_SAFE_INTEGER) fail(`${key} 数值约束无效`)
    }
    if (schema.unit && !numeric) fail('unit 仅支持数值')
    if (schema.precision !== undefined && (!Number.isInteger(schema.precision) || (schema.precision as number) < 0 || (schema.precision as number) > 12)) fail('precision 必须为 0..12')
    if (schema.multiple_of !== undefined && (schema.multiple_of as number) <= 0) fail('multiple_of 必须为正')
    if (schema.minimum !== undefined && schema.maximum !== undefined && (schema.minimum as number) > (schema.maximum as number)) fail('minimum 大于 maximum')
    for (const [min, max, type] of [['min_length', 'max_length', 'string'], ['min_items', 'max_items', 'array']]) {
      for (const key of [min!, max!]) if (own(schema, key) && (schema.type !== type || !Number.isSafeInteger(schema[key]) || (schema[key] as number) < 0)) fail(`${key} 无效`)
      if (schema[min!] !== undefined && schema[max!] !== undefined && (schema[min!] as number) > (schema[max!] as number)) fail('最小长度大于最大长度')
    }
    const format = schema.format || ''
    if (!['', 'unix-ms', 'int64', 'uint64', 'node-id', 'base64'].includes(String(format))) fail('unsupported variable format')
    if (format === 'unix-ms' ? schema.type !== 'integer' : format !== '' && schema.type !== 'string') fail('format 与类型不符')
    if (!['', '^[0-9a-f]{32}$', '^[1-9][0-9]*$'].includes(String(schema.pattern || ''))) fail('unsupported variable pattern')
    if (schema.pattern && schema.type !== 'string') fail('pattern 需要 string')
    if (schema.content_type && (format !== 'base64' || !/^[!#$%&'*+.^_`|~0-9A-Za-z-]+\/[!#$%&'*+.^_`|~0-9A-Za-z-]+(?:[ \t]*;[ \t]*[!#$%&'*+.^_`|~0-9A-Za-z-]+=(?:[!#$%&'*+.^_`|~0-9A-Za-z-]+|"[^"\r\n]*"))*$/.test(String(schema.content_type)))) fail('content_type 无效')
    if (own(schema, 'enum')) {
      if (schema.type !== 'string' || !Array.isArray(schema.enum) || schema.enum.length > 64) fail('enum 无效或超过 64 项')
      for (let i = 0; i < schema.enum.length; i++) {
        const entry = schema.enum[i]
        text(entry, 1024)
        if (i && compareUTF8(String(schema.enum[i - 1]), String(entry)) >= 0) fail('enum 必须排序且唯一')
        validateString(schema as unknown as DataSchemaDefinition, entry as string, false)
      }
    }
    if (schema.type === 'object') {
      if (own(schema, 'items')) fail('object 不支持 items')
      if (own(schema, 'properties') && !Array.isArray(schema.properties)) fail('properties 需要数组')
      const names = new Set<string>()
      for (const item of (schema.properties || []) as unknown[]) {
        const property = record(item)
        keys(property, ['name', 'required', 'schema'])
        text(property.name, 128, true)
        if (own(property, 'required') && typeof property.required !== 'boolean') fail('required 需要 boolean')
        if (names.has(property.name as string) || ++properties > 128) fail('properties 重复或超过 128 项')
        names.add(property.name as string)
        visit(property.schema, depth + 1)
      }
      if (own(schema, 'additional_properties')) visit(schema.additional_properties, depth + 1)
    } else if (schema.type === 'array') {
      if (own(schema, 'properties') || own(schema, 'additional_properties')) fail('array 不支持对象属性')
      visit(schema.items, depth + 1)
    } else if (['properties', 'items', 'additional_properties'].some((key) => own(schema, key))) fail('标量不支持结构字段')
  }
  visit(value, 1)
  if (encodedBytes(value) > 65536) fail('definition 超过 64 KiB')
}

function decimal(raw: string): [bigint, bigint] {
  const [mantissa = '', power = '0'] = raw.toLowerCase().split('e')
  const exponent = Number(power)
  if (!Number.isInteger(exponent) || exponent < -324 || exponent > 308) fail('数字指数超限')
  const fraction = mantissa.split('.')[1]?.length || 0
  const numerator = BigInt(mantissa.replace('.', ''))
  const scale = fraction - exponent
  return scale >= 0 ? [numerator, 10n ** BigInt(scale)] : [numerator * 10n ** BigInt(-scale), 1n]
}
function validateNumber(schema: DataSchemaDefinition, value: number, raw = String(value)) {
  if (!Number.isFinite(value) || Math.abs(value) > Number.MAX_SAFE_INTEGER) fail('数字超出 JS 安全范围')
  const [n, d] = decimal(raw)
  if ((n < 0 ? -n : n) > BigInt(Number.MAX_SAFE_INTEGER) * d || value === 0 && n !== 0n) fail('数值精度溢出')
  if (schema.type === 'integer' && n % d !== 0n) fail('需要整数')
  for (const [bound, minimum] of [[schema.minimum, true], [schema.maximum, false]] as const) {
    if (bound === undefined) continue
    const [bn, bd] = decimal(String(bound))
    if (minimum ? n * bd < bn * d : n * bd > bn * d) fail('数字越界')
  }
  if (schema.multiple_of !== undefined) { const [sn, sd] = decimal(String(schema.multiple_of)); if ((n * sd) % (d * sn) !== 0n) fail('数字不符合 multiple_of 步长') }
  if (schema.precision !== undefined && (n * 10n ** BigInt(schema.precision)) % d !== 0n) fail('数字超过 precision')
}
function validateString(schema: DataSchemaDefinition, value: string, checkEnum = true) {
  for (let i = 0; i < value.length; i++) {
    const code = value.charCodeAt(i)
    if (code >= 0xd800 && code <= 0xdbff) { const next = value.charCodeAt(++i); if (!(next >= 0xdc00 && next <= 0xdfff)) fail('无效 Unicode') }
    else if (code >= 0xdc00 && code <= 0xdfff) fail('无效 Unicode')
  }
  const length = bytes(value)
  if (schema.min_length !== undefined && length < schema.min_length || schema.max_length !== undefined && length > schema.max_length) fail('文本 UTF-8 字节长度越界')
  if (checkEnum && schema.enum?.length && !schema.enum.includes(value)) fail('文本不在 enum 中')
  if (schema.pattern === '^[0-9a-f]{32}$') hex(value)
  if (schema.pattern === '^[1-9][0-9]*$' && !/^[1-9][0-9]*$/.test(value)) fail('文本不符合十进制 pattern')
  if (schema.format === 'node-id') parseRevision(value)
  if (schema.format === 'int64' || schema.format === 'uint64') {
    if (!/^(?:0|-?[1-9][0-9]*)$/.test(value) || value.length > 20) fail('需要规范十进制文本')
    const number = BigInt(value)
    if (schema.format === 'int64' ? number < -9223372036854775808n || number > 9223372036854775807n : number < 0n || number > 18446744073709551615n) fail('十进制文本越界')
  }
  if (schema.format === 'base64') {
    try { if (btoa(atob(value)) !== value) fail('需要规范 base64 文本') } catch { fail('需要规范 base64 文本') }
  }
}
export function validateVariableValue(schema: DataSchemaDefinition, value: unknown, numbers?: Map<string, string>, path: string[] = [], depth = 1): asserts value is JSONValue {
  if (depth > 12) fail('value 超过 12 层')
  if (value === null) { if (schema.nullable || schema.type === 'null') return; fail('null 不被定义允许') }
  if (schema.type === 'boolean' && typeof value === 'boolean') return
  if ((schema.type === 'number' || schema.type === 'integer') && typeof value === 'number') return validateNumber(schema, value, numbers?.get(JSON.stringify(path)))
  if (schema.type === 'string' && typeof value === 'string') return validateString(schema, value)
  if (schema.type === 'array' && Array.isArray(value)) {
    if (schema.min_items !== undefined && value.length < schema.min_items || schema.max_items !== undefined && value.length > schema.max_items) fail('数组长度越界')
    value.forEach((item, index) => validateVariableValue(schema.items!, item, numbers, [...path, String(index)], depth + 1)); return
  }
  if (schema.type === 'object' && value && typeof value === 'object' && !Array.isArray(value)) {
    const object = record(value)
    const known = new Map(schema.properties?.map((property) => [property.name, property]))
    for (const property of schema.properties || []) if (property.required && !own(object, property.name)) fail(`缺少必填字段 ${property.name}`)
    for (const [key, child] of Object.entries(object)) {
      text(key, 128, true)
      const definition = known.get(key)?.schema || schema.additional_properties
      if (!definition) fail(`未知对象字段 ${key}`)
      validateVariableValue(definition, child, numbers, [...path, key], depth + 1)
    }
    return
  }
  fail(`值需要 ${schema.type}`)
}
export function validateVariableJSON(schema: DataSchemaDefinition, raw: string): unknown {
  validateVariableDefinition(schema)
  const parsed = parseBoundedJSON(raw)
  validateVariableValue(schema, parsed.value, parsed.numbers)
  return parsed.value
}
export function sameValue(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true
  if (!left || !right || typeof left !== 'object' || typeof right !== 'object' || Array.isArray(left) !== Array.isArray(right)) return false
  const a = Object.keys(left), b = Object.keys(right)
  return a.length === b.length && a.every((key) => own(right, key) && sameValue((left as Record<string, unknown>)[key], (right as Record<string, unknown>)[key]))
}
export function parseVariableEnvelope(value: unknown, previous?: VariableValueV1, limit = MAX_VARIABLE_BYTES, requireDefinition = true): VariableValueV1 {
  const envelope = record(value)
  if (encodedBytes(value) > limit) fail('Variable envelope 超过大小上限')
  keys(envelope, ['version', 'instance_id', 'revision', 'definition', 'value', 'status', 'source_time_unix_ms', 'observed_at_unix_ms', 'error'])
  if (envelope.version !== 1) fail('unsupported Variable envelope version')
  hex(envelope.instance_id); parseRevision(envelope.revision)
  if (!Number.isSafeInteger(envelope.observed_at_unix_ms) || own(envelope, 'source_time_unix_ms') && !Number.isSafeInteger(envelope.source_time_unix_ms)) fail('时间必须为 JS 安全整数')
  if (!['ready', 'stale', 'unavailable'].includes(String(envelope.status))) fail('Variable status 无效')
  if (envelope.status === 'unavailable' ? own(envelope, 'value') : !own(envelope, 'value')) fail('Variable status 与 value 不一致')
  if (own(envelope, 'error')) {
    if (envelope.status === 'ready') fail('ready 不允许 error')
    const error = record(envelope.error)
    keys(error, ['code', 'message']); text(error.code, 128, true); text(error.message, 1024)
  }
  const definition = own(envelope, 'definition') ? envelope.definition : previous?.instance_id === envelope.instance_id ? previous?.definition : undefined
  if (definition === undefined && !requireDefinition) return envelope as unknown as VariableValueV1
  validateVariableDefinition(definition)
  if (own(envelope, 'value')) validateVariableValue(definition, envelope.value)
  return { ...envelope, definition } as VariableValueV1
}
export function parseVariableEvent(event: ResourceEvent, resource: ResourceDescriptor, previous?: VariableValueV1) {
  if (event.owner_node_id !== resource.id.owner_node_id || event.resource_name !== resource.id.name || event.capability !== 'subscribe' || event.schema !== VARIABLE_VALUE_SCHEMA) fail('Variable event 身份/schema 不符')
  if (event.kind !== 'snapshot' && event.kind !== 'data') fail('unsupported Variable event kind')
  const result = parseVariableEnvelope(event.value, previous, variablePayloadLimit(resource, 'subscribe'))
  if (event.kind === 'data' && previous?.instance_id !== result.instance_id) fail('data 属于未知实例，需要新快照')
  if (event.revision !== result.revision) fail('event revision 与 envelope 不一致')
  return result
}

function compareUTF8(a: string, b: string): number {
  const left = new TextEncoder().encode(a), right = new TextEncoder().encode(b)
  for (let i = 0; i < Math.min(left.length, right.length); i++) if (left[i] !== right[i]) return left[i]! - right[i]!
  return left.length - right.length
}
export function parseVariableEnvelopeJSON(raw: string): VariableValueV1 {
  const parsed = parseBoundedJSON(raw)
  validateRawDefinitionNumbers(parsed.numbers)
  validateRawEnvelopeNumbers(parsed.numbers)
  const envelope = parseVariableEnvelope(parsed.value)
  if (Object.prototype.hasOwnProperty.call(envelope, 'value')) validateVariableValue(envelope.definition!, envelope.value, parsed.numbers, ['value'])
  return envelope
}

export function validateVariableWireJSON(raw: string, kind: 'value' | 'write'): void {
  const { value, numbers } = parseBoundedJSON(raw)
  validateRawDefinitionNumbers(numbers)
  validateRawEnvelopeNumbers(numbers)
  const object = record(value)
  if (encodedBytes(value) > MAX_VARIABLE_BYTES) fail('Variable wire 超过上限')
  if (kind === 'value') {
    const envelope = parseVariableEnvelope(value, undefined, MAX_VARIABLE_BYTES, false)
    if (envelope.definition && own(envelope, 'value')) validateVariableValue(envelope.definition, envelope.value, numbers, ['value'])
  } else {
    keys(object, ['version', 'expected_instance_id', 'expected_revision', 'value'])
    if (object.version !== 3 || !own(object, 'value')) fail('unsupported Variable write')
    hex(object.expected_instance_id); parseRevision(object.expected_revision)
  }
}

export function validateRawDefinitionNumbers(numbers: Map<string, string>): void {
  for (const [path, raw] of numbers) {
    const names = JSON.parse(path) as string[]
    if (names.includes('definition') && ['minimum', 'maximum', 'multiple_of'].includes(names.at(-1)!)) validateNumber({ type: 'number' }, Number(raw), raw)
    if (names.includes('definition') && ['precision', 'min_length', 'max_length', 'min_items', 'max_items'].includes(names.at(-1)!) && !/^-?(0|[1-9][0-9]*)$/.test(raw)) fail('definition 整数约束不能含小数或指数')
  }
}

export function validateRawEnvelopeNumbers(numbers: Map<string, string>, prefix: string[] = []): void {
  for (const key of ['version', 'observed_at_unix_ms', 'source_time_unix_ms']) {
    const raw = numbers.get(JSON.stringify([...prefix, key]))
    if (raw !== undefined && (!/^-?(0|[1-9][0-9]*)$/.test(raw) || !Number.isSafeInteger(Number(raw)))) fail(`${key} 必须为安全 JSON 整数`)
  }
}

export function parseVariableOperationPayload(raw: string): { schema?: string; payload: unknown } {
  const parsed = parseBoundedJSON(raw, MAX_VARIABLE_BYTES + 4096)
  const operation = record(parsed.value)
  if (operation.schema !== VARIABLE_VALUE_SCHEMA) return operation as { schema?: string; payload: unknown }
  validateRawDefinitionNumbers(parsed.numbers)
  validateRawEnvelopeNumbers(parsed.numbers, ['payload'])
  const payload = parseVariableEnvelope(operation.payload)
  if (own(payload, 'value')) validateVariableValue(payload.definition!, payload.value, parsed.numbers, ['payload', 'value'])
  return { schema: VARIABLE_VALUE_SCHEMA, payload }
}
