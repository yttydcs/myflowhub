import type { DataSchemaDefinition } from '../generated/variable-contract.generated'
import { descriptorError } from '../variable/protocol'
import type { ResourceDescriptor } from '../types'
import type { DataSchema, SchemaResolution, SchemaUse } from './schema'
import { resolveResourceSchema } from './schema'

export type PaneDensity = 'compact' | 'normal' | 'expanded'

export type RendererChoice = {
  id: string
  label: string
  description: string
}

const typeHints = new Set([ 'mfh.stream', 'mfh.topic', 'mfh.command', 'mfh.file', 'mfh.collection', 'mfh.auto'])

export type RendererSettingsValidation = { settings?: Record<string, never>; reason?: string }

// Phase 1 renderers have no persistent tuning knobs. Keeping the allowlist
// empty prevents drafts, payloads, or arbitrary provider data entering a View.
export function validateRendererSettings(rendererID: string, value: unknown): RendererSettingsValidation {
  if (value === undefined || value === null) return {}
  if (typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0) return {}
  return { reason: `${rendererID} 不接受持久化 settings` }
}

const specialized: Record<string, RendererChoice> = {
  'mfh.catalog.v2': { id: 'mfh.structured.catalog.v1', label: '资源表格', description: '按资源、类型与能力浏览目录' },
  'mfh.management.topology.v1': { id: 'mfh.structured.topology.v1', label: '节点表格', description: '展示节点、父级与角色' },
  'mfh.management.health.v1': { id: 'mfh.structured.health.v1', label: '健康状态', description: '突出运行状态与组件检查' },
  'mfh.file.transfers.v1': { id: 'mfh.structured.transfers.v1', label: '传输列表', description: '展示文件传输状态与进度' },
  'mfh.file.progress.v1': { id: 'mfh.event.progress.v1', label: '传输进度', description: '按传输展示进度事件' },
}

function scalarChoices(schema: DataSchema, writable: boolean): RendererChoice[] {
  if (schema.type === 'boolean') return [
    { id: writable ? 'mfh.variable.switch.v2' : 'mfh.variable.status.v2', label: writable ? '开关' : '状态', description: writable ? '暂存布尔修改后应用' : '以明确文字展示布尔状态' },
    ...(!writable ? [{ id: 'mfh.variable.text.v2', label: '文本', description: '以文本展示值' }] : []),
  ]
  if (schema.type === 'integer' || schema.type === 'number') {
    const choices: RendererChoice[] = [
      { id: 'mfh.variable.number.v2', label: writable ? '数字输入' : '数值', description: writable ? '使用有约束的数字输入' : '突出显示数值' },
    ]
    if (schema.minimum !== undefined && schema.maximum !== undefined) {
      choices.push({ id: writable ? 'mfh.variable.slider.v2' : 'mfh.variable.progress.v2', label: writable ? '滑块' : '进度', description: '使用 provider 声明的上下限和步长' })
    }
    return choices
  }
  if (schema.type === 'string') {
    if (schema.enum?.length) return [
      { id: writable ? 'mfh.variable.select.v2' : 'mfh.variable.status.v2', label: writable ? '选项' : '状态', description: '使用 provider 声明的枚举' },
      { id: 'mfh.variable.text.v2', label: '文本', description: '以普通文本展示' },
    ]
    return [
      { id: 'mfh.variable.text.v2', label: writable ? '单行文本' : '文本', description: '紧凑文本展示或编辑' },
      { id: 'mfh.variable.multiline.v2', label: '多行文本', description: '用于较长的自然语言文本' },
      { id: 'mfh.variable.code.v2', label: '代码文本', description: '使用等宽多行编辑器' },
    ]
  }
  return []
}

function unique(choices: RendererChoice[]): RendererChoice[] {
  const seen = new Set<string>()
  return choices.filter((choice) => !seen.has(choice.id) && Boolean(seen.add(choice.id)))
}

export function resourceSchemaUse(resource: ResourceDescriptor): SchemaUse {
  if (resource.type === 'mfh.stream' || resource.type === 'mfh.topic') return 'event'
  if (resource.type === 'mfh.collection') return 'operation-output'
  if (resource.type === 'mfh.command') return 'operation-input'
  return 'value'
}

export function rendererChoices(resource: ResourceDescriptor, resolution: SchemaResolution): RendererChoice[] {
  if (resource.type === 'mfh.collection') {
    const list = resource.capabilities.find((candidate) => candidate.name === 'list')
    if (list?.input_schema === 'mfh.collection.list-request.v1' && list.output_schema === 'mfh.collection.page.v1') {
      return [{ id: 'mfh.collection.browser.v1', label: 'Collection', description: '分页浏览并查看 Collection 成员' }]
    }
    return [{ id: 'mfh.collection.unsupported.v1', label: 'Collection descriptor', description: 'list capability schema 与 Collection v1 不兼容' }]
  }
  if (resource.type === 'mfh.file') return [{ id: 'mfh.file.transfer.v1', label: '文件传输', description: '选择本地文件并通过 session 上传' }]
  if (resource.type === 'mfh.command') {
    const result = resolution.status === 'resolved'
      ? [{ id: 'mfh.operation.form.v1', label: '表单', description: '按 provider schema 生成字段' }]
      : []
    return [...result, { id: 'mfh.operation.json.v1', label: 'Advanced JSON', description: '直接编辑 operation payload' }]
  }
  if (resource.type === 'mfh.stream' || resource.type === 'mfh.topic') {
    const first = resolution.status === 'resolved' ? specialized[resolution.schemaID] : undefined
    return unique([
      ...(first ? [first] : []),
      { id: 'mfh.event.timeline.v1', label: '时间线', description: '按到达顺序查看结构化事件' },
      { id: 'mfh.event.table.v1', label: '表格', description: '按公共字段比较事件' },
      { id: 'mfh.event.log.v1', label: '日志', description: '紧凑查看原始事件' },
    ])
  }
  if (resource.type === 'mfh.variable') return selectVariableRenderer(resource).choices
  return [{ id: 'mfh.unknown.raw.v1', label: 'Descriptor', description: '保留未知 Resource 的完整描述' }]
}

export type RendererSelection = {
  choices: RendererChoice[]
  selected: RendererChoice
  fallbackReason?: string
  resolution: SchemaResolution
}

export function selectResourceRenderer(resource: ResourceDescriptor, requested?: string): RendererSelection {
  if (resource.type === 'mfh.variable') {
    const selection = selectVariableRenderer(resource, undefined, requested)
    return { ...selection, resolution: { status: 'missing', reason: '等待实例快照 definition' } }
  }
  const resolution = resolveResourceSchema(resource, resourceSchemaUse(resource))
  const choices = rendererChoices(resource, resolution)
  const providerHint = resource.presentation?.renderer
  const candidate = requested && !typeHints.has(requested) ? requested : providerHint && !typeHints.has(providerHint) ? providerHint : undefined
  const selected = choices.find((choice) => choice.id === candidate) || choices[0]!
  return {
    choices,
    selected,
    resolution,
    fallbackReason: candidate && selected.id !== candidate ? `已保存的显示方式 ${candidate} 与当前 schema/capability 不兼容，暂时使用 ${selected.label}` : undefined,
  }
}

export function selectVariableRenderer(resource: ResourceDescriptor, definition?: DataSchemaDefinition, requested?: string): Pick<RendererSelection, 'choices' | 'selected' | 'fallbackReason'> {
  const reason = descriptorError(resource)
  const choices: RendererChoice[] = reason
    ? [{ id: 'mfh.variable.unsupported.v2', label: 'Unsupported Variable', description: reason }]
    : !definition
      ? [{ id: 'mfh.variable.auto.v2', label: '自动', description: '随实例快照选择控件' }]
      : [...scalarChoices(definition as DataSchema, resource.capabilities.some((item) => item.name === 'write') && !definition.read_only),
          ...(['object', 'array', 'null'].includes(definition.type) ? [{ id: 'mfh.variable.structured.v2', label: '结构化', description: '按实例定义展示与编辑' }] : [])]
  const candidate = requested || resource.presentation?.renderer
  const selected = choices.find((choice) => choice.id === candidate) || choices[0]!
  return { choices, selected, fallbackReason: reason || (candidate && candidate !== 'mfh.variable.auto.v2' && selected.id !== candidate ? `显示方式 ${candidate} 不受当前 definition 支持，使用 ${selected.label}` : undefined) }
}
