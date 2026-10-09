import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parseRevision, validateVariableDefinition, validateVariableJSON, validateVariableWireJSON } from './protocol'

const vectors = JSON.parse(readFileSync(resolve(process.cwd(), '../../../protocol/testdata/variable_validation_vectors.json'), 'utf8')) as {
  version: number
  definitions: { name: string; definition: unknown; valid: boolean }[]
  values: { name: string; definition: unknown; value: unknown; valid: boolean }[]
  revisions: { revision: string; valid: boolean }[]
  wire: { name: string; kind: 'value' | 'write'; json: string; valid: boolean }[]
}
const succeeds = (operation: () => unknown) => { try { operation(); return true } catch { return false } }
describe('canonical Go/TS Variable vectors', () => {
  it('uses the shared current vector version', () => expect(vectors.version).toBe(1))
  it.each(vectors.definitions)('definition $name', ({ definition, valid }) => expect(succeeds(() => validateVariableDefinition(definition))).toBe(valid))
  it.each(vectors.values)('value $name', ({ definition, value, valid }) => expect(succeeds(() => {
    validateVariableDefinition(definition)
    validateVariableJSON(definition, JSON.stringify(value))
  })).toBe(valid))
  it.each(vectors.revisions)('revision $revision', ({ revision, valid }) => expect(succeeds(() => parseRevision(revision))).toBe(valid))
  it.each(vectors.wire)('wire $name', ({ kind, json, valid }) => expect(succeeds(() => validateVariableWireJSON(json, kind))).toBe(valid))
})
