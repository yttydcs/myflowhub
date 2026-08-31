export const COLLECTION_LIST_REQUEST_SCHEMA = 'mfh.collection.list-request.v1'
export const COLLECTION_MEMBER_REQUEST_SCHEMA = 'mfh.collection.member-request.v1'
export const COLLECTION_MEMBER_SCHEMA = 'mfh.collection.member.v1'
export const COLLECTION_PAGE_SCHEMA = 'mfh.collection.page.v1'
export const FILESYSTEM_READ_REQUEST_SCHEMA = 'mfh.filesystem.read-request.v1'
export const FILESYSTEM_CONTENT_SCHEMA = 'mfh.filesystem.content.v1'
export const COLLECTION_PAGE_LIMIT = 64
export const MAX_COLLECTION_PAGE_MEMBERS = 256
export const MAX_VISIBLE_COLLECTION_MEMBERS = 1024
export const MAX_FILESYSTEM_READ_BYTES = 131_072

const MAX_KEY_LENGTH = 1024
const MAX_KIND_LENGTH = 128
const MAX_LABEL_LENGTH = 255
const MAX_SCHEMA_LENGTH = 255
const MAX_CONTENT_TYPE_LENGTH = 127
const MAX_CURSOR_LENGTH = 1024
const MAX_CAPABILITIES = 64
const MAX_ATTRIBUTES = 64
const FILESYSTEM_CONTENT_FIELDS = new Set(['version', 'key', 'content_type', 'encoding', 'data', 'size', 'modified_unix_ms', 'revision'])

export type CollectionMember = {
  key: string
  kind: string
  label: string
  content_type?: string
  schema?: string
  capabilities: string[]
  attributes?: Record<string, string>
}

export type CollectionPage = {
  version: 1
  revision: number
  parent: string
  members: CollectionMember[]
  next_cursor: string
}

export type SafeFilesystemContent = {
  key: string
  contentType: string
  modifiedUnixMS: number
  revision: string
  size: number
  warnings: string[]
} & (
  | { mode: 'json'; value: unknown; text: string }
  | { mode: 'text'; text: string }
  | { mode: 'image'; bytes: Uint8Array }
  | { mode: 'unsupported'; reason: string }
)

function objectValue(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${label} 必须是对象`)
  return value as Record<string, unknown>
}

function boundedString(value: unknown, label: string, maximum: number, required = false): string {
  if (value === undefined && !required) return ''
  if (typeof value !== 'string') throw new Error(`${label} 必须是文本`)
  if (required && !value) throw new Error(`${label} 不能为空`)
  if (new TextEncoder().encode(value).byteLength > maximum) throw new Error(`${label} 超过 ${maximum} bytes`)
  return value
}

function positiveInteger(value: unknown, label: string): number {
  if (!Number.isSafeInteger(value) || (value as number) <= 0) throw new Error(`${label} 必须是安全范围内的正整数`)
  return value as number
}

function compareUTF8(left: string, right: string): number {
  const encoder = new TextEncoder()
  const leftBytes = encoder.encode(left)
  const rightBytes = encoder.encode(right)
  const common = Math.min(leftBytes.length, rightBytes.length)
  for (let index = 0; index < common; index += 1) {
    if (leftBytes[index] !== rightBytes[index]) return leftBytes[index]! - rightBytes[index]!
  }
  return leftBytes.length - rightBytes.length
}

export function parseCollectionMember(value: unknown): CollectionMember {
  const record = objectValue(value, 'Collection member')
  const capabilitiesValue = record.capabilities ?? []
  if (!Array.isArray(capabilitiesValue) || capabilitiesValue.length > MAX_CAPABILITIES) {
    throw new Error(`member capabilities 必须是不超过 ${MAX_CAPABILITIES} 项的数组`)
  }
  const capabilities = capabilitiesValue.map((capability, index) => boundedString(capability, `capabilities[${index}]`, MAX_KIND_LENGTH, true))
  if (capabilities.some((capability) => !/^[\p{L}\p{N}][\p{L}\p{N}._-]*$/u.test(capability))) {
    throw new Error('member capabilities 包含无效 identifier')
  }
  for (let index = 1; index < capabilities.length; index += 1) {
    if (compareUTF8(capabilities[index - 1]!, capabilities[index]!) >= 0) throw new Error('member capabilities 必须严格排序且唯一')
  }
  let attributes: Record<string, string> | undefined
  if (record.attributes !== undefined) {
    const source = objectValue(record.attributes, 'member attributes')
    const entries = Object.entries(source)
    if (entries.length > MAX_ATTRIBUTES) throw new Error(`member attributes 超过 ${MAX_ATTRIBUTES} 项`)
    attributes = Object.fromEntries(entries.map(([key, attribute]) => {
      boundedString(key, 'attribute key', 64, true)
      return [key, boundedString(attribute, `attribute ${key}`, MAX_KEY_LENGTH)]
    }))
  }
  const contentType = boundedString(record.content_type, 'member content_type', MAX_CONTENT_TYPE_LENGTH)
  const schema = boundedString(record.schema, 'member schema', MAX_SCHEMA_LENGTH)
  return {
    key: boundedString(record.key, 'member key', MAX_KEY_LENGTH, true),
    kind: boundedString(record.kind, 'member kind', MAX_KIND_LENGTH, true),
    label: boundedString(record.label, 'member label', MAX_LABEL_LENGTH, true),
    ...(contentType ? { content_type: contentType } : {}),
    ...(schema ? { schema } : {}),
    capabilities,
    ...(attributes ? { attributes } : {}),
  }
}

export function parseCollectionPage(value: unknown, expectedParent: string): CollectionPage {
  const record = objectValue(value, 'Collection page')
  if (record.version !== 1) throw new Error('Collection page version 必须为 1')
  const parent = boundedString(record.parent, 'Collection page parent', MAX_KEY_LENGTH)
  if (parent !== expectedParent) throw new Error('Collection page parent 与当前目录不一致')
  if (!Array.isArray(record.members) || record.members.length > MAX_COLLECTION_PAGE_MEMBERS) {
    throw new Error(`Collection page members 必须是不超过 ${MAX_COLLECTION_PAGE_MEMBERS} 项的数组`)
  }
  const members = record.members.map(parseCollectionMember)
  for (let index = 1; index < members.length; index += 1) {
    if (compareUTF8(members[index - 1]!.key, members[index]!.key) >= 0) throw new Error('Collection page members 必须按 key 严格排序且唯一')
  }
  return {
    version: 1,
    revision: positiveInteger(record.revision, 'Collection page revision'),
    parent,
    members,
    next_cursor: boundedString(record.next_cursor, 'Collection page next_cursor', MAX_CURSOR_LENGTH),
  }
}

export function appendCollectionPage(current: CollectionMember[], page: CollectionPage): CollectionMember[] {
  if (current.length && page.members.length && compareUTF8(current.at(-1)!.key, page.members[0]!.key) >= 0) {
    throw new Error('下一页成员与当前列表重复或顺序无效')
  }
  if (current.length + page.members.length > MAX_VISIBLE_COLLECTION_MEMBERS) {
    throw new Error(`当前浏览最多保留 ${MAX_VISIBLE_COLLECTION_MEMBERS} 个成员`)
  }
  return [...current, ...page.members]
}

function normalizedContentType(value: string): string {
  return value.trim().toLocaleLowerCase()
}

function mediaType(value: string): string {
  return normalizedContentType(value).split(';', 1)[0]!.trim()
}

function decodeStrictBase64(value: string): Uint8Array {
  if (value.length % 4 !== 0 || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value)) {
    throw new Error('filesystem content 包含无效 base64')
  }
  let binary: string
  try { binary = atob(value) } catch { throw new Error('filesystem content 包含无效 base64') }
  if (btoa(binary) !== value) throw new Error('filesystem content base64 不是规范编码')
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}

function decodeUTF8(bytes: Uint8Array): string {
  try { return new TextDecoder('utf-8', { fatal: true }).decode(bytes) } catch { throw new Error('filesystem content 不是有效 UTF-8') }
}

function encodeStrictUTF8(value: string): Uint8Array {
  const bytes = new TextEncoder().encode(value)
  if (decodeUTF8(bytes) !== value) throw new Error('filesystem content 不是有效 UTF-8')
  return bytes
}

export function parseFilesystemContent(
  value: unknown,
  expected: { key: string; contentType?: string; revision?: string },
): SafeFilesystemContent {
  const record = objectValue(value, 'Filesystem content')
  for (const key of Object.keys(record)) {
    if (!FILESYSTEM_CONTENT_FIELDS.has(key)) throw new Error(`Filesystem content 包含未知字段 ${key}`)
  }
  if (record.version !== 1) throw new Error('Filesystem content version 必须为 1')
  const key = boundedString(record.key, 'Filesystem content key', MAX_KEY_LENGTH, true)
  if (key !== expected.key) throw new Error('Filesystem content key 与所选成员不一致')
  const contentType = boundedString(record.content_type, 'Filesystem content content_type', MAX_CONTENT_TYPE_LENGTH, true)
  const warnings: string[] = []
  if (expected.contentType && normalizedContentType(contentType) !== normalizedContentType(expected.contentType)) {
    warnings.push(`读取结果 content_type ${contentType} 与成员提示 ${expected.contentType} 不一致；已按读取结果安全展示`)
  }
  const encoding = boundedString(record.encoding, 'Filesystem content encoding', 16, true)
  if (encoding !== 'utf-8' && encoding !== 'base64') throw new Error(`不支持的 filesystem encoding ${encoding}`)
  if (typeof record.data !== 'string') throw new Error('Filesystem content data 必须是文本')
  const data = boundedString(record.data, 'Filesystem content data', 174_764)
  const size = Number(record.size)
  if (!Number.isSafeInteger(size) || size < 0 || size > MAX_FILESYSTEM_READ_BYTES) throw new Error('Filesystem content size 超出有界读取范围')
  const modifiedUnixMS = Number(record.modified_unix_ms)
  if (!Number.isSafeInteger(modifiedUnixMS) || modifiedUnixMS < 0) throw new Error('Filesystem content modified_unix_ms 无效')
  const revision = boundedString(record.revision, 'Filesystem content revision', 128, true)
  if (expected.revision && revision !== expected.revision) throw new Error('Filesystem content revision 与请求不一致')
  const bytes = encoding === 'base64' ? decodeStrictBase64(data) : encodeStrictUTF8(data)
  if (bytes.byteLength !== size) throw new Error('Filesystem content size 与实际内容不一致')

  const base = { key, contentType, modifiedUnixMS, revision, size, warnings }
  const type = mediaType(contentType)
  if (['image/png', 'image/jpeg', 'image/gif', 'image/webp'].includes(type)) {
    if (encoding !== 'base64') return { ...base, mode: 'unsupported', reason: '栅格图像必须使用 base64 编码' }
    return { ...base, mode: 'image', bytes }
  }
  if (type === 'image/svg+xml') return { ...base, mode: 'unsupported', reason: 'SVG 内容不会嵌入或执行' }
  if (type === 'application/json' || type.endsWith('+json')) {
    const text = encoding === 'utf-8' ? data : decodeUTF8(bytes)
    try { return { ...base, mode: 'json', value: JSON.parse(text), text } }
    catch { return { ...base, mode: 'text', text, warnings: [...warnings, 'JSON 解析失败，已按转义文本显示'] } }
  }
  if (type.startsWith('text/')) {
    const text = encoding === 'utf-8' ? data : decodeUTF8(bytes)
    return { ...base, mode: 'text', text, warnings: type === 'text/html' ? [...warnings, 'HTML 已按转义文本显示，不会执行'] : warnings }
  }
  return { ...base, mode: 'unsupported', reason: `不支持内联显示 ${contentType || 'binary'}；内容未执行或伪装为文本` }
}
