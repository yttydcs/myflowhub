import { useEffect } from 'react'

// StatusJSON is local Host state. This never polls remote trees or catalogs.
export function useConnectionMonitor(enabled: boolean, check: (current: () => boolean) => Promise<void>) {
  useEffect(() => {
    if (!enabled) return
    let stopped = false
    let timer: ReturnType<typeof setTimeout>
    const tick = async () => {
      await check(() => !stopped)
      if (!stopped) timer = setTimeout(() => void tick(), 1000)
    }
    timer = setTimeout(() => void tick(), 1000)
    return () => { stopped = true; clearTimeout(timer) }
  }, [enabled, check])
}
