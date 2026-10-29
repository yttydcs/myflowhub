const entries = new Set<symbol>()
const listeners = new Set<() => void>()
export const hasUnsavedChanges = () => entries.size > 0
export const subscribeUnsavedChanges = (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener) } }
export function setUnsavedChanges(owner: symbol, dirty: boolean) {
  const before = entries.size > 0
  if (dirty) entries.add(owner)
  else entries.delete(owner)
  if (before !== (entries.size > 0)) listeners.forEach(listener => listener())
}
