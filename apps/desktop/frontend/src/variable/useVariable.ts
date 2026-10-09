import { createContext, useContext, useEffect, useState } from 'react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor } from '../types'
import { acquireVariable, type VariableController, type VariableSessionState } from './controller'

export const VariableProfileContext = createContext('standalone')
const empty: VariableSessionState = { dirty: false, conflict: false, blocked: '', error: '', observation: '', live: false, loading: true, writing: false }
export function useVariable(api: DesktopAPI, resource: ResourceDescriptor) {
  const profile = useContext(VariableProfileContext)
  const key = JSON.stringify([profile, resource.id, resource.type_version, resource.capabilities, resource.limits])
  const [session, setSession] = useState<{ key: string; controller?: VariableController; state: VariableSessionState }>({ key, state: empty })
  useEffect(() => {
    let release: (() => void) | undefined
    let unsubscribe: (() => void) | undefined
    try {
      const acquired = acquireVariable(api, profile, resource)
      release = acquired.release
      const update = () => setSession({ key, controller: acquired.controller, state: acquired.controller.getSnapshot() })
      unsubscribe = acquired.controller.subscribe(update)
      update()
    } catch (error) { setSession({ key, state: { ...empty, loading: false, error: String(error), blocked: String(error) } }) }
    return () => { unsubscribe?.(); release?.() }
  }, [api, key])
  return session.key === key ? session : { key, state: empty }
}
