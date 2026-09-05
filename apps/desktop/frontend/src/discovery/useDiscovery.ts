import { useEffect, useMemo, useSyncExternalStore } from 'react'
import type { DesktopAPI } from '../api'
import { DiscoveryController } from './controller'

export function useDiscovery(api: DesktopAPI) {
  const controller = useMemo(() => new DiscoveryController(api), [api])
  const snapshot = useSyncExternalStore(controller.subscribe, controller.getSnapshot)
  useEffect(() => () => controller.reset(), [controller])
  return { controller, snapshot }
}
