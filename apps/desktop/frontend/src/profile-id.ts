const PROFILE_ID_ATTEMPTS = 64

export type RandomBytes = (length: number) => Uint8Array

function secureRandomBytes(length: number): Uint8Array {
  if (!globalThis.crypto?.getRandomValues) throw new Error('当前环境无法安全生成 Profile ID。')
  return globalThis.crypto.getRandomValues(new Uint8Array(length))
}

export function generateProfileID(existingIDs: Iterable<string>, randomBytes: RandomBytes = secureRandomBytes): string {
  const existing = new Set(existingIDs)
  for (let attempt = 0; attempt < PROFILE_ID_ATTEMPTS; attempt += 1) {
    const bytes = randomBytes(8)
    if (bytes.length !== 8) throw new Error('Profile ID 随机源返回了错误长度。')
    const suffix = [...bytes].map((value) => value.toString(16).padStart(2, '0')).join('')
    const candidate = `profile-${suffix}`
    if (!existing.has(candidate)) return candidate
  }
  throw new Error('无法生成唯一的 Profile ID，请删除不用的 Profile 后重试。')
}
