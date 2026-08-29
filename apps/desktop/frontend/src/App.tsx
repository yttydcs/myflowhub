import { useCallback, useEffect, useMemo, useState } from 'react'
import { DndContext, type DragEndEvent } from '@dnd-kit/core'
import { Boxes, Layers3, Moon, Plus, RefreshCw, Settings2, Sun } from 'lucide-react'
import { api as productionApi, type DesktopAPI, type PreparedProfile } from './api'
import { Explorer } from './components/Explorer'
import { Inspector } from './components/Inspector'
import { LoginScreen } from './components/LoginScreen'
import { Settings as DesktopSettings } from './components/Settings'
import { ViewManager, Workspace } from './components/Workspace'
import { Button } from './components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './components/ui/tabs'
import { errorText } from './lib/utils'
import { defaultUIPreferences, loadUIPreferences, saveUIPreferences, type Theme, type UIPreferences } from './preferences'
import { addWidget, nextWidgetID } from './store'
import type { ConnectionStatus, Profile, ResourceDescriptor, Settings, Topology, ViewDefinition, WorkspaceSelection } from './types'

const emptyTopology: Topology = { version: 1, epoch: 1, nodes: [] }

function newView(index = 1): ViewDefinition {
  return { id: `view-${index}`, name: `工作视图 ${index}`, revision: 0, widgets: [] }
}

export function App({ api = productionApi }: { api?: DesktopAPI }) {
  const [settings, setSettings] = useState<Settings>()
  const [status, setStatus] = useState<ConnectionStatus>({ state: 'signed_out' })
  const [topology, setTopology] = useState<Topology>(emptyTopology)
  const [resources, setResources] = useState<ResourceDescriptor[]>([])
  const [selection, setSelection] = useState<WorkspaceSelection>(null)
  const [views, setViews] = useState<ViewDefinition[]>([])
  const [view, setView] = useState<ViewDefinition>(newView())
  const [activeContent, setActiveContent] = useState<'workspace' | 'settings'>('workspace')
  const [preferences, setPreferences] = useState<UIPreferences>(defaultUIPreferences())
  const [dirty, setDirty] = useState(false)
  const [busy, setBusy] = useState(false)
  const [savingView, setSavingView] = useState(false)
  const [error, setError] = useState('')

  const activeProfile = useMemo(
    () => settings?.profiles.find((profile) => profile.id === settings.active_profile_id),
    [settings],
  )
  const confirmDiscard = useCallback(
    () => !dirty || window.confirm('当前视图有未保存的更改。放弃这些更改并继续吗？'),
    [dirty],
  )

  useEffect(() => {
    const guard = (event: BeforeUnloadEvent) => {
      if (!dirty) return
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', guard)
    return () => window.removeEventListener('beforeunload', guard)
  }, [dirty])

  useEffect(() => {
    if (!activeProfile) {
      setPreferences(defaultUIPreferences())
      return
    }
    const loaded = loadUIPreferences(activeProfile.id)
    setPreferences(loaded.value)
    if (loaded.warning) setError((current) => current || loaded.warning || '')
  }, [activeProfile?.id])

  useEffect(() => {
    document.documentElement.dataset.theme = preferences.theme
    document.documentElement.style.colorScheme = preferences.theme
  }, [preferences.theme])

  const loadViews = useCallback(async () => {
    const document = await api.views()
    setViews(document.views)
    setView(document.views[0] || newView(1))
    setDirty(false)
  }, [api])

  const refreshPlatform = useCallback(async (profile: Profile, waitForAutoConnect = true) => {
    setError('')
    let nextStatus = await api.status()
    for (let attempt = 0; waitForAutoConnect && profile.auto_connect && attempt < 60 && (nextStatus.state === 'disconnected' || nextStatus.state === 'connecting'); attempt += 1) {
      await new Promise((resolve) => window.setTimeout(resolve, 250))
      nextStatus = await api.status()
    }
    setStatus(nextStatus)
    if (nextStatus.state !== 'connected') {
      setTopology(emptyTopology)
      setResources([])
      setSelection(null)
      return
    }
    const nextTopology = await api.topology(profile.parent_node_id)
    const catalogs = await Promise.allSettled(nextTopology.nodes.map((node) => api.catalog(node.node_id)))
    const nextResources = catalogs.flatMap((result) => result.status === 'fulfilled' ? result.value.resources : [])
    setTopology(nextTopology)
    setResources(nextResources)
    setSelection((current) => {
      if (current?.kind === 'node') {
        return nextTopology.nodes.some((node) => node.node_id === current.node.node_id) ? current : null
      }
      if (current?.kind === 'resource') {
        return nextResources.some((resource) => (
          resource.id.owner_node_id === current.resource.id.owner_node_id
          && resource.id.name === current.resource.id.name
        )) ? current : null
      }
      return current
    })
    const failures = catalogs.filter((result): result is PromiseRejectedResult => result.status === 'rejected')
    const firstFailure = failures[0]
    if (firstFailure) setError(`已连接，但有 ${failures.length} 个节点的资源目录加载失败：${errorText(firstFailure.reason)}`)
  }, [api])

  useEffect(() => {
    void (async () => {
      try {
        const nextSettings = await api.settings()
        setSettings(nextSettings)
        const profile = nextSettings.profiles.find((item) => item.id === nextSettings.active_profile_id)
        if (profile) await Promise.all([refreshPlatform(profile), loadViews()])
      } catch (current) {
        setError(errorText(current))
      }
    })()
  }, [api, loadViews, refreshPlatform])

  function updatePreferences(patch: Partial<Omit<UIPreferences, 'version'>>) {
    const next: UIPreferences = { ...preferences, ...patch, version: 1 }
    setPreferences(next)
    if (!activeProfile) return
    const saveError = saveUIPreferences(activeProfile.id, next)
    if (saveError) setError(saveError)
  }

  const login = async (profile: Profile, permit: string) => {
    if (!confirmDiscard()) return false
    setBusy(true)
    setError('')
    try {
      const saved = await api.login(profile, permit)
      const next = await api.settings()
      setSettings(next)
      const nextProfile = next.profiles.find((item) => item.id === next.active_profile_id) || saved
      await Promise.all([refreshPlatform(nextProfile), loadViews()])
      setActiveContent('workspace')
      setSelection(null)
      return true
    } catch (current) {
      setError(errorText(current))
      return false
    } finally {
      setBusy(false)
    }
  }

  const prepareProfile = async (profile: Profile): Promise<PreparedProfile | undefined> => {
    setBusy(true)
    setError('')
    try {
      const prepared = await api.prepareProfile(profile)
      setSettings(await api.settings())
      return prepared
    } catch (current) {
      setError(errorText(current))
      return undefined
    } finally {
      setBusy(false)
    }
  }

  const saveProfile = async (profile: Profile) => {
    if (!confirmDiscard()) return false
    setBusy(true)
    setError('')
    try {
      const saved = await api.saveProfile(profile)
      const next = await api.settings()
      setSettings(next)
      setSelection(null)
      await Promise.all([refreshPlatform(saved, false), loadViews()])
      return true
    } catch (current) {
      setError(errorText(current))
      return false
    } finally {
      setBusy(false)
    }
  }

  const switchProfile = async (profileID: string) => {
    if (profileID === activeProfile?.id) return
    if (!confirmDiscard()) return
    setBusy(true)
    setError('')
    try {
      await api.switchProfile(profileID)
      const next = await api.settings()
      setSettings(next)
      const profile = next.profiles.find((item) => item.id === profileID)
      if (profile) await Promise.all([refreshPlatform(profile), loadViews()])
      setSelection(null)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const deleteProfile = async (profileID: string) => {
    if (profileID === activeProfile?.id && !confirmDiscard()) return
    setBusy(true)
    setError('')
    try {
      await api.deleteProfile(profileID, `DELETE ${profileID}`)
      const next = await api.settings()
      setSettings(next)
      if (profileID === activeProfile?.id) {
        setStatus({ state: 'signed_out' })
        setTopology(emptyTopology)
        setResources([])
        setViews([])
        setView(newView())
        setDirty(false)
      }
      setSelection(null)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const connect = async () => {
    setBusy(true)
    setError('')
    try {
      await api.connect()
      if (activeProfile) await refreshPlatform(activeProfile)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const disconnect = async () => {
    setBusy(true)
    setError('')
    try {
      await api.disconnect()
      setStatus(await api.status())
      setTopology(emptyTopology)
      setResources([])
      setSelection(null)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const refresh = async () => {
    if (!activeProfile) return
    setBusy(true)
    try {
      await refreshPlatform(activeProfile)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const changeView = (next: ViewDefinition) => {
    setView(next)
    setDirty(true)
  }
  const addResource = (resource: ResourceDescriptor) => {
    setActiveContent('workspace')
    changeView({ ...view, widgets: addWidget(view.widgets, resource, nextWidgetID(resource, view.widgets)) })
  }
  const dragEnd = (event: DragEndEvent) => {
    if (event.over?.id !== 'workspace-drop') return
    const resource = event.active.data.current?.resource as ResourceDescriptor | undefined
    if (resource) addResource(resource)
  }
  const saveView = async () => {
    setBusy(true)
    setSavingView(true)
    setError('')
    try {
      const saved = await api.saveView(view)
      setView(saved)
      setViews((current) => [...current.filter((item) => item.id !== saved.id), saved].sort((left, right) => left.id.localeCompare(right.id)))
      setDirty(false)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setSavingView(false)
      setBusy(false)
    }
  }
  const createView = () => {
    if (!confirmDiscard()) return
    let index = views.length + 1
    while (views.some((item) => item.id === `view-${index}`)) index += 1
    setView(newView(index))
    setDirty(true)
    setActiveContent('workspace')
  }
  const removeView = async (target: ViewDefinition) => {
    if (!window.confirm(`删除视图“${target.name}”？此操作不可恢复。`)) return
    if (target.revision === 0) {
      setView(views[0] || newView())
      setDirty(false)
      return
    }
    try {
      await api.deleteView(target.id, target.revision)
      const next = views.filter((item) => item.id !== target.id)
      setViews(next)
      setView(next[0] || newView(1))
      setDirty(false)
    } catch (current) {
      setError(errorText(current))
    }
  }

  if (!settings) {
    return <main className="boot-screen"><span className="brand-mark" aria-hidden="true">M</span><p>正在打开资源工作区…</p>{error && <p className="form-error" role="alert">{error}</p>}</main>
  }
  if (!activeProfile) {
    return (
      <LoginScreen
        settings={settings}
        busy={busy}
        error={error}
        theme={preferences.theme}
        onThemeChange={(theme) => updatePreferences({ theme })}
        onPrepare={prepareProfile}
        onLogin={login}
        onDelete={deleteProfile}
      />
    )
  }

  const inspectorVisible = activeContent === 'workspace' && selection !== null
  return (
    <DndContext onDragEnd={dragEnd}>
      <main className="app-shell">
        <a className="skip-link" href="#resource-workspace">跳到资源工作区</a>
        <header className="topbar">
          <h1 className="sr-only">MyFlowHub 资源工作区</h1>
          <div className="brand-lockup"><span className="brand-mark small" aria-hidden="true">M</span><strong>MyFlowHub</strong></div>
          <div className="topbar-actions">
            <Button variant="ghost" size="icon" aria-label="刷新节点树" disabled={busy} onClick={() => void refresh()}><RefreshCw aria-hidden="true" size={15} /></Button>
            <Button variant="ghost" size="icon" aria-label={preferences.theme === 'light' ? '切换到深色主题' : '切换到浅色主题'} onClick={() => updatePreferences({ theme: preferences.theme === 'light' ? 'dark' : 'light' })}>
              {preferences.theme === 'light' ? <Moon aria-hidden="true" size={15} /> : <Sun aria-hidden="true" size={15} />}
            </Button>
          </div>
        </header>
        {error && <div className="error-banner" role="alert" aria-live="polite">{error}<button aria-label="关闭错误消息" onClick={() => setError('')}>×</button></div>}
        <div className={`app-body ${activeContent === 'settings' ? 'show-settings' : ''} ${inspectorVisible ? 'show-inspector' : ''}`}>
          <aside className="sidebar">
            <Tabs defaultValue="explorer" className="sidebar-tabs">
              <TabsList><TabsTrigger value="explorer"><Boxes aria-hidden="true" size={14} />资源</TabsTrigger><TabsTrigger value="views"><Layers3 aria-hidden="true" size={14} />视图</TabsTrigger></TabsList>
              <TabsContent className="tabs-content" value="explorer">
                <Explorer
                  topology={topology}
                  resources={resources}
                  selection={selection}
                  expandedNodeIDs={preferences.expanded_node_ids}
                  focusedNodeID={preferences.focused_node_id}
                  onExpandedNodeIDsChange={(expandedNodeIDs) => updatePreferences({ expanded_node_ids: expandedNodeIDs })}
                  onFocusedNodeIDChange={(focusedNodeID) => updatePreferences({ focused_node_id: focusedNodeID })}
                  onSelect={setSelection}
                  onAdd={addResource}
                />
              </TabsContent>
              <TabsContent className="tabs-content" value="views">
                <ViewManager
                  views={views}
                  activeID={view.id}
                  onOpen={(target) => {
                    if (!confirmDiscard()) return
                    setView(target)
                    setDirty(false)
                    setActiveContent('workspace')
                  }}
                  onCreate={createView}
                  onDelete={(target) => void removeView(target)}
                />
              </TabsContent>
            </Tabs>
            <button className="sidebar-profile" onClick={() => setActiveContent('settings')} aria-label={`打开 ${activeProfile.name} 的设置`}>
              <span className="profile-avatar" aria-hidden="true">{activeProfile.name.slice(0, 1).toUpperCase()}</span>
              <span><strong>{activeProfile.name}</strong><small><span className={`status-dot ${status.state}`} aria-hidden="true" />{connectionLabel(status.state)}</small></span>
              <Settings2 aria-hidden="true" size={15} />
            </button>
          </aside>

          <div className="main-stack">
            <nav className="content-tabs" role="tablist" aria-label="工作区标签">
              <button role="tab" aria-selected={activeContent === 'workspace'} className={activeContent === 'workspace' ? 'is-active' : ''} onClick={() => setActiveContent('workspace')}><Layers3 aria-hidden="true" size={13} />{view.name}{dirty && <span className="tab-dirty" aria-label="未保存">●</span>}</button>
              <button role="tab" aria-selected={activeContent === 'settings'} className={activeContent === 'settings' ? 'is-active' : ''} onClick={() => setActiveContent('settings')}><Settings2 aria-hidden="true" size={13} />设置</button>
              <button className="new-tab" onClick={createView} aria-label="新建视图"><Plus aria-hidden="true" size={14} /></button>
            </nav>
            {activeContent === 'settings'
              ? (
                <DesktopSettings
                  settings={settings}
                  activeProfile={activeProfile}
                  status={status}
                  theme={preferences.theme}
                  busy={busy}
                  onConnect={connect}
                  onDisconnect={disconnect}
                  onSwitchProfile={switchProfile}
                  onSaveProfile={saveProfile}
                  onDeleteProfile={deleteProfile}
                  onThemeChange={(theme: Theme) => updatePreferences({ theme })}
                />
              )
              : (
                <Workspace
                  id="resource-workspace"
                  api={api}
                  resources={resources}
                  view={view}
                  dirty={dirty}
                  saving={savingView}
                  onChange={changeView}
                  onSave={() => void saveView()}
                />
              )}
          </div>

          {inspectorVisible && (
            <Inspector
              api={api}
              selection={selection}
              resources={resources}
              onClose={() => setSelection(null)}
              onAdd={addResource}
              onFocusNode={(nodeID) => updatePreferences({
                focused_node_id: nodeID,
                expanded_node_ids: [...new Set([...(preferences.expanded_node_ids || []), nodeID])],
              })}
            />
          )}
        </div>
      </main>
    </DndContext>
  )
}

function connectionLabel(state: ConnectionStatus['state']): string {
  return {
    signed_out: '未登录',
    disconnected: '已断开',
    connecting: '连接中',
    connected: '已连接',
    failed: '连接失败',
    stopped: '已停止',
  }[state]
}
