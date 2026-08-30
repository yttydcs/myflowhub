import generatedSchemas from '../generated/resource-schemas.generated.json'
import type { CapabilityDescriptor, ResourceDescriptor } from '../types'

export type DataSchemaType = 'null' | 'boolean' | 'integer' | 'number' | 'string' | 'object' | 'array'

export type DataSchemaProperty = {
  name: string
  required?: boolean
  schema: DataSchema
}

export type DataSchema = {
  id?: string
  type: DataSchemaType
  title?: string
  description?: string
  format?: string
  unit?: string
  precision?: number
  minimum?: number
  maximum?: number
  multiple_of?: number
  min_length?: number
  max_length?: number
  pattern?: string
  min_items?: number
  max_items?: number
  enum?: string[]
  read_only?: boolean
  write_only?: boolean
  sensitive?: boolean
  properties?: DataSchemaProperty[]
  items?: DataSchema
  additional_properties?: DataSchema
}

export type SchemaResolution =
  | { status: 'resolved'; schema: DataSchema; schemaID: string; source: string }
  | { status: 'missing' | 'unsupported' | 'invalid'; schemaID?: string; reason: string }

export type SchemaProvider = {
  id: string
  resolve(schemaID: string): SchemaResolution | undefined
}

type SchemaDocument = { version: number; schemas: DataSchema[] }

const MAX_SCHEMA_DEPTH = 12
const MAX_SCHEMA_PROPERTIES = 128
const MAX_SCHEMA_ENUM = 64
const SCHEMA_KEYS = new Set([
  'id', 'type', 'title', 'description', 'format', 'unit', 'precision', 'minimum', 'maximum', 'multiple_of',
  'min_length', 'max_length', 'pattern', 'min_items', 'max_items', 'enum', 'read_only', 'write_only',
  'sensitive', 'properties', 'items', 'additional_properties',
])

function schemaDefinitionError(schema: DataSchema, depth = 1, propertyCount = { value: 0 }): string | undefined {
  if (!schema || typeof schema !== 'object' || Array.isArray(schema)) return 'schema 必须是对象'
  if (depth > MAX_SCHEMA_DEPTH) return `schema 超过 ${MAX_SCHEMA_DEPTH} 层`
  const unsupported = Object.keys(schema).find((key) => !SCHEMA_KEYS.has(key))
  if (unsupported) return `不支持关键字 ${unsupported}`
  if (!['null', 'boolean', 'integer', 'number', 'string', 'object', 'array'].includes(schema.type)) return `不支持类型 ${String(schema.type)}`
  if (schema.precision !== undefined && (!Number.isInteger(schema.precision) || schema.precision < 0 || schema.precision > 12)) return 'precision 必须是 0..12 的整数'
  if (schema.minimum !== undefined && !Number.isFinite(schema.minimum)) return 'minimum 必须是有限数字'
  if (schema.maximum !== undefined && !Number.isFinite(schema.maximum)) return 'maximum 必须是有限数字'
  if (schema.minimum !== undefined && schema.maximum !== undefined && schema.minimum > schema.maximum) return 'minimum 大于 maximum'
  if (schema.multiple_of !== undefined && (!Number.isFinite(schema.multiple_of) || schema.multiple_of <= 0)) return 'multiple_of 必须为正数'
  for (const [name, minimum, maximum] of [['length', schema.min_length, schema.max_length], ['items', schema.min_items, schema.max_items]] as const) {
    if (minimum !== undefined && (!Number.isInteger(minimum) || minimum < 0)) return `min_${name} 必须是非负整数`
    if (maximum !== undefined && (!Number.isInteger(maximum) || maximum < 0)) return `max_${name} 必须是非负整数`
    if (minimum !== undefined && maximum !== undefined && minimum > maximum) return `min_${name} 大于 max_${name}`
  }
  if (schema.enum && (!Array.isArray(schema.enum) || schema.enum.some((value) => typeof value !== 'string'))) return 'enum 必须是文本数组'
  if ((schema.enum?.length || 0) > MAX_SCHEMA_ENUM) return `enum 超过 ${MAX_SCHEMA_ENUM} 项`
  if (schema.enum && new Set(schema.enum).size !== schema.enum.length) return 'enum 包含重复值'
  if (schema.pattern) {
    if (schema.type !== 'string') return 'pattern 只能用于 string'
    try { new RegExp(schema.pattern) } catch { return 'pattern 不是有效正则表达式' }
  }
  if (schema.sensitive && !schema.write_only) return 'sensitive 字段必须同时为 write_only'
  if (schema.type === 'object') {
    if (schema.items) return 'object 不能声明 items'
    if (schema.properties !== undefined && !Array.isArray(schema.properties)) return 'properties 必须是数组'
    const names = new Set<string>()
    for (const property of schema.properties || []) {
      const unsupportedProperty = Object.keys(property).find((key) => !['name', 'required', 'schema'].includes(key))
      if (unsupportedProperty) return `对象属性不支持关键字 ${unsupportedProperty}`
      if (!property.name || names.has(property.name)) return `对象属性 ${property.name || '(empty)'} 无效或重复`
      if (!property.schema) return `对象属性 ${property.name} 缺少 schema`
      names.add(property.name)
      propertyCount.value += 1
      if (propertyCount.value > MAX_SCHEMA_PROPERTIES) return `schema 超过 ${MAX_SCHEMA_PROPERTIES} 个属性`
      const child = schemaDefinitionError(property.schema, depth + 1, propertyCount)
      if (child) return `${property.name}: ${child}`
    }
    if (schema.additional_properties) {
      const child = schemaDefinitionError(schema.additional_properties, depth + 1, propertyCount)
      if (child) return `additional_properties: ${child}`
    }
    return undefined
  }
  if (schema.type === 'array') {
    if (!schema.items) return 'array 缺少 items'
    if (schema.properties || schema.additional_properties) return 'array 不能声明对象属性'
    return schemaDefinitionError(schema.items, depth + 1, propertyCount)
  }
  if (schema.properties || schema.items || schema.additional_properties) return '标量 schema 不能声明结构字段'
  return undefined
}

function createBuiltinProvider(): SchemaProvider {
  const document = generatedSchemas as SchemaDocument
  const definitions = new Map<string, DataSchema>()
  const invalid = new Map<string, string>()
  if (document.version === 1 && Array.isArray(document.schemas)) {
    for (const schema of document.schemas) {
      if (!schema.id || definitions.has(schema.id)) continue
      const reason = schemaDefinitionError(schema)
      if (reason) invalid.set(schema.id, reason)
      else definitions.set(schema.id, schema)
    }
  }
  return {
    id: 'mfh.builtin.v1',
    resolve(schemaID) {
      const reason = invalid.get(schemaID)
      if (reason) return { status: 'invalid', schemaID, reason: `内置 schema 无效：${reason}` }
      const schema = definitions.get(schemaID)
      return schema ? { status: 'resolved', schema, schemaID, source: 'mfh.builtin.v1' } : undefined
    },
  }
}

const builtinProvider = createBuiltinProvider()

export function resolveSchema(schemaID?: string, providers: SchemaProvider[] = [builtinProvider]): SchemaResolution {
  if (!schemaID) return { status: 'missing', reason: 'Resource 没有声明 schema ID' }
  for (const provider of providers) {
    const result = provider.resolve(schemaID)
    if (result?.status === 'resolved') {
      const reason = schemaDefinitionError(result.schema)
      return reason ? { status: 'invalid', schemaID, reason: `${provider.id} 返回无效 schema：${reason}` } : result
    }
    if (result) return result
  }
  return { status: 'unsupported', schemaID, reason: `尚未安装 ${schemaID} 的声明式 schema` }
}

export function capability(resource: ResourceDescriptor, name: string): CapabilityDescriptor | undefined {
  return resource.capabilities.find((candidate) => candidate.name === name)
}

export type SchemaUse = 'value' | 'operation-input' | 'operation-output' | 'event'

export function resourceSchemaID(resource: ResourceDescriptor, use: SchemaUse, capabilityName?: string): string | undefined {
  if (use === 'value') {
    return capability(resource, 'read')?.output_schema || capability(resource, 'subscribe')?.event_schema
  }
  if (use === 'event') return capability(resource, 'subscribe')?.event_schema
  const descriptor = capability(resource, capabilityName || (resource.type === 'mfh.topic' ? 'publish' : 'invoke'))
  return use === 'operation-input' ? descriptor?.input_schema : descriptor?.output_schema
}

export function resolveResourceSchema(resource: ResourceDescriptor, use: SchemaUse, capabilityName?: string): SchemaResolution {
  return resolveSchema(resourceSchemaID(resource, use, capabilityName))
}

export type DataValidationError = { path: string; message: string }

export function validateDataValue(schema: DataSchema, value: unknown, path = '$', depth = 1): DataValidationError[] {
  if (depth > MAX_SCHEMA_DEPTH) return [{ path, message: '值超过可验证深度' }]
  if (value === null) return schema.type === 'null' ? [] : [{ path, message: `需要 ${schema.type}` }]
  const errors: DataValidationError[] = []
  switch (schema.type) {
    case 'null':
      errors.push({ path, message: '需要 null' })
      break
    case 'boolean':
      if (typeof value !== 'boolean') errors.push({ path, message: '需要布尔值' })
      break
    case 'integer':
    case 'number': {
      if (typeof value !== 'number' || !Number.isFinite(value) || (schema.type === 'integer' && !Number.isInteger(value))) {
        errors.push({ path, message: schema.type === 'integer' ? '需要整数' : '需要数字' })
        break
      }
      if (schema.minimum !== undefined && value < schema.minimum) errors.push({ path, message: `不能小于 ${schema.minimum}` })
      if (schema.maximum !== undefined && value > schema.maximum) errors.push({ path, message: `不能大于 ${schema.maximum}` })
      if (schema.multiple_of !== undefined) {
        const quotient = value / schema.multiple_of
        if (Math.abs(quotient - Math.round(quotient)) > 1e-9) errors.push({ path, message: `步长必须为 ${schema.multiple_of}` })
      }
      break
    }
    case 'string': {
      if (typeof value !== 'string') {
        errors.push({ path, message: '需要文本' })
        break
      }
      if (schema.min_length !== undefined && value.length < schema.min_length) errors.push({ path, message: `至少 ${schema.min_length} 个字符` })
      if (schema.max_length !== undefined && value.length > schema.max_length) errors.push({ path, message: `最多 ${schema.max_length} 个字符` })
      if (schema.enum?.length && !schema.enum.includes(value)) errors.push({ path, message: '值不在允许选项中' })
      if (schema.pattern && !new RegExp(schema.pattern).test(value)) errors.push({ path, message: '文本格式不符合要求' })
      if (schema.format === 'json' && value) {
        try { JSON.parse(value) } catch { errors.push({ path, message: '需要有效 JSON 文本' }) }
      }
      break
    }
    case 'object': {
      if (typeof value !== 'object' || Array.isArray(value)) {
        errors.push({ path, message: '需要对象' })
        break
      }
      const record = value as Record<string, unknown>
      const known = new Set((schema.properties || []).map((property) => property.name))
      for (const property of schema.properties || []) {
        if (!(property.name in record)) {
          if (property.required) errors.push({ path: `${path}.${property.name}`, message: '必填' })
          continue
        }
        errors.push(...validateDataValue(property.schema, record[property.name], `${path}.${property.name}`, depth + 1))
      }
      if (schema.additional_properties) {
        for (const [name, child] of Object.entries(record)) {
          if (!known.has(name)) errors.push(...validateDataValue(schema.additional_properties, child, `${path}.${name}`, depth + 1))
        }
      }
      break
    }
    case 'array': {
      if (!Array.isArray(value)) {
        errors.push({ path, message: '需要数组' })
        break
      }
      if (schema.min_items !== undefined && value.length < schema.min_items) errors.push({ path, message: `至少 ${schema.min_items} 项` })
      if (schema.max_items !== undefined && value.length > schema.max_items) errors.push({ path, message: `最多 ${schema.max_items} 项` })
      if (schema.items) value.forEach((item, index) => errors.push(...validateDataValue(schema.items!, item, `${path}[${index}]`, depth + 1)))
      break
    }
  }
  return errors
}

export function defaultDataValue(schema: DataSchema): unknown {
  if (schema.enum?.length) return schema.enum[0]
  switch (schema.type) {
    case 'null': return null
    case 'boolean': return false
    case 'integer':
    case 'number': return schema.minimum ?? 0
    case 'string': return ''
    case 'array': return []
    case 'object': {
      const result: Record<string, unknown> = {}
      for (const property of schema.properties || []) {
        if (property.required && !property.schema.write_only) result[property.name] = defaultDataValue(property.schema)
      }
      return result
    }
  }
}

export function schemaProperty(schema: DataSchema, name: string): DataSchema | undefined {
  return schema.properties?.find((property) => property.name === name)?.schema
}
