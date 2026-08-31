import { describe, expect, it } from 'vitest'
import {
  appendCollectionPage,
  MAX_COLLECTION_PAGE_MEMBERS,
  parseCollectionMember,
  parseCollectionPage,
  parseFilesystemContent,
} from './collection'

const member = (key: string, overrides: Record<string, unknown> = {}) => ({
  key,
  kind: 'file',
  label: key,
  capabilities: ['get', 'read'],
  ...overrides,
})

describe('collection response boundaries', () => {
  it('accepts a deterministic bounded page and appends the next page', () => {
    const first = parseCollectionPage({ version: 1, revision: 7, parent: '', members: [member('a')], next_cursor: 'opaque-1' }, '')
    const next = parseCollectionPage({ version: 1, revision: 7, parent: '', members: [member('b')], next_cursor: '' }, '')
    expect(appendCollectionPage(first.members, next).map((candidate) => candidate.key)).toEqual(['a', 'b'])
  })

  it.each([
    [{ version: 2, revision: 1, parent: '', members: [] }, /version/],
    [{ version: 1, revision: 0, parent: '', members: [] }, /revision/],
    [{ version: 1, revision: 1, parent: 'other', members: [] }, /parent/],
    [{ version: 1, revision: 1, parent: '', members: [member('b'), member('a')] }, /严格排序/],
    [{ version: 1, revision: 1, parent: '', members: [member('a'), member('a')] }, /严格排序/],
    [{ version: 1, revision: 1, parent: '', members: Array.from({ length: MAX_COLLECTION_PAGE_MEMBERS + 1 }, (_, index) => member(String(index).padStart(3, '0'))) }, /256/],
  ])('rejects a malformed page %#', (value, message) => {
    expect(() => parseCollectionPage(value, '')).toThrow(message as RegExp)
  })

  it('rejects unsorted, duplicate and oversized member metadata', () => {
    expect(() => parseCollectionMember(member('a', { capabilities: ['read', 'get'] }))).toThrow(/严格排序/)
    expect(() => parseCollectionMember(member('a', { capabilities: ['read', 'read'] }))).toThrow(/严格排序/)
    expect(() => parseCollectionMember(member('a', { capabilities: ['_invalid'] }))).toThrow(/identifier/)
    expect(() => parseCollectionMember(member('a', { attributes: Object.fromEntries(Array.from({ length: 65 }, (_, index) => [`a${index}`, 'x'])) }))).toThrow(/64/)
  })
})

describe('safe filesystem content decoding', () => {
  const content = (overrides: Record<string, unknown> = {}) => ({
    version: 1,
    key: 'readme',
    content_type: 'text/plain; charset=utf-8',
    encoding: 'utf-8',
    data: '<script>alert(1)</script>',
    size: 25,
    modified_unix_ms: 1,
    revision: 'r1',
    ...overrides,
  })

  it('keeps HTML as escaped text and parses JSON only after validation', () => {
    expect(parseFilesystemContent(content({ content_type: 'text/html' }), { key: 'readme' })).toMatchObject({
      mode: 'text', text: '<script>alert(1)</script>', warnings: [expect.stringContaining('不会执行')],
    })
    expect(parseFilesystemContent(content({ content_type: 'application/json', data: '{"ok":true}', size: 11 }), { key: 'readme' })).toMatchObject({ mode: 'json', value: { ok: true } })
    expect(parseFilesystemContent(content({ content_type: 'application/json', data: '{bad}', size: 5 }), { key: 'readme' })).toMatchObject({ mode: 'text', warnings: [expect.stringContaining('解析失败')] })
  })

  it('accepts only canonical base64 raster images and rejects invalid data', () => {
    const image = parseFilesystemContent(content({ content_type: 'image/png', encoding: 'base64', data: 'iVBORw==', size: 4 }), { key: 'readme', contentType: 'image/png' })
    expect(image).toMatchObject({ mode: 'image', size: 4 })
    expect(() => parseFilesystemContent(content({ encoding: 'base64', data: '!!!!', size: 3 }), { key: 'readme' })).toThrow(/base64/)
  })

  it('never treats SVG or other binary as executable media and detects mismatches', () => {
    expect(parseFilesystemContent(content({ content_type: 'image/svg+xml', encoding: 'base64', data: 'PHN2Zz4=', size: 5 }), { key: 'readme' })).toMatchObject({ mode: 'unsupported', reason: expect.stringContaining('SVG') })
    expect(parseFilesystemContent(content({ content_type: 'application/octet-stream', encoding: 'base64', data: 'AA==', size: 1 }), { key: 'readme' })).toMatchObject({ mode: 'unsupported' })
    expect(() => parseFilesystemContent(content(), { key: 'other' })).toThrow(/key/)
    expect(parseFilesystemContent(content(), { key: 'readme', contentType: 'application/octet-stream' })).toMatchObject({
      mode: 'text', warnings: [expect.stringContaining('不一致')],
    })
    expect(() => parseFilesystemContent(content(), { key: 'readme', revision: 'r2' })).toThrow(/revision/)
    expect(() => parseFilesystemContent(content({ size: 24 }), { key: 'readme' })).toThrow(/size/)
    expect(() => parseFilesystemContent(content({ data: undefined, size: 0 }), { key: 'readme' })).toThrow(/data/)
    expect(() => parseFilesystemContent(content({ unexpected: true }), { key: 'readme' })).toThrow(/未知字段/)
  })

  it('rejects invalid UTF-8 represented by an unpaired surrogate', () => {
    expect(() => parseFilesystemContent(content({ data: '\ud800', size: 3 }), { key: 'readme' })).toThrow(/UTF-8/)
  })

  it('measures protocol string bounds in UTF-8 bytes and rejects unsafe revisions', () => {
    expect(() => parseCollectionMember(member('a', { label: '测'.repeat(86) }))).toThrow(/255 bytes/)
    expect(parseCollectionPage({ version: 1, revision: Number.MAX_SAFE_INTEGER, parent: '', members: [] }, '')).toMatchObject({ revision: Number.MAX_SAFE_INTEGER })
    expect(() => parseCollectionPage({ version: 1, revision: Number.MAX_SAFE_INTEGER + 1, parent: '', members: [] }, '')).toThrow(/安全范围/)
  })
})
