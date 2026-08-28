import { useCallback, useEffect, useMemo, useState } from 'react'
import { DndContext, type DragEndEvent } from '@dnd-kit/core'
import { Boxes, CloudOff, Layers3, RefreshCw, UserPlus, Wifi } from 'lucide-react'
import { api as productionApi, type DesktopAPI } from './api'
import { LoginScreen } from './components/LoginScreen'
import { Explorer } from './components/Explorer'
import { ViewManager, Workspace } from './components/Workspace'
import { Button } from './components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './components/ui/tabs'
import { errorText } from './lib/utils'
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
  const [dirty, setDirty] = useState(false)
  const [busy, setBusy] = useState(false)
  const [savingView, setSavingView] = useState(false)
  const [error, setError] = useState('')
  const [showLogin, setShowLogin] = useState(false)

  const activeProfile = useMemo(() => settings?.profiles.find((profile) => profile.id === settings.active_profile_id), [settings])
  const confirmDiscard = useCallback(() => !dirty || window.confirm('当前视图有未保存的更改。放弃这些更改并继续吗？'), [dirty])

  useEffect(() => {
    const guard = (event: BeforeUnloadEvent) => {
      if (!dirty) return
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', guard)
    return () => window.removeEventListener('beforeunload', guard)
  }, [dirty])

  const loadViews = useCallback(async () => {
    const document = await api.views()
    setViews(document.views)
    setView(document.views[0] || newView(1))
    setDirty(false)
  }, [api])

  const refreshPlatform = useCallback(async (profile: Profile) => {
    setError('')
    let nextStatus = await api.status()
    for (let attempt = 0; profile.auto_connect && attempt < 60 && (nextStatus.state === 'disconnected' || nextStatus.state === 'connecting'); attempt++) {
      await new Promise((resolve) => window.setTimeout(resolve, 250))
      nextStatus = await api.status()
    }
    setStatus(nextStatus)
    if (nextStatus.state !== 'connected') return
    const nextTopology = await api.topology(profile.parent_node_id)
    const catalogs = await Promise.allSettled(nextTopology.nodes.map((node) => api.catalog(node.node_id)))
    setTopology(nextTopology)
    setResources(catalogs.flatMap((result) => result.status === 'fulfilled' ? result.value.resources : []))
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
      } catch (current) { setError(errorText(current)) }
    })()
  }, [api, loadViews, refreshPlatform])

  const login = async (profile: Profile, permit: string) => {
    if (!confirmDiscard()) return false
    setBusy(true); setError('')
    try {
      await api.login(profile, permit)
      const next = await api.settings()
      setSettings(next)
      setShowLogin(false)
      await Promise.all([refreshPlatform(profile), loadViews()])
      return true
    } catch (current) { setError(errorText(current)); return false }
    finally { setBusy(false) }
  }

  const switchProfile = async (profileID: string) => {
    if (profileID === activeProfile?.id) { setShowLogin(false); return }
    if (!confirmDiscard()) return
    setBusy(true); setError('')
    try {
      await api.switchProfile(profileID)
      const next = await api.settings()
      setSettings(next)
      const profile = next.profiles.find((item) => item.id === profileID)
      if (profile) await Promise.all([refreshPlatform(profile), loadViews()])
      setSelection(null)
      setShowLogin(false)
    } catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }

  const deleteProfile = async (profileID: string) => {
    if (profileID === activeProfile?.id && !confirmDiscard()) return
    setBusy(true); setError('')
    try {
      await api.deleteProfile(profileID, `DELETE ${profileID}`)
      const next = await api.settings()
      setSettings(next)
      setSelection(null)
    } catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }

  const connect = async () => {
    setBusy(true); setError('')
    try {
      await api.connect()
      if (activeProfile) await refreshPlatform(activeProfile)
    } catch (current) { setError(errorText(current)) }
    finally { setBusy(false) }
  }

  const changeView = (next: ViewDefinition) => { setView(next); setDirty(true) }
  const addResource = (resource: ResourceDescriptor) => changeView({ ...view, widgets: addWidget(view.widgets, resource, nextWidgetID(resource, view.widgets)) })
  const dragEnd = (event: DragEndEvent) => {
    if (event.over?.id !== 'workspace-drop') return
    const resource = event.active.data.current?.resource as ResourceDescriptor | undefined
    if (resource) addResource(resource)
  }
  const save = async () => {
    setBusy(true); setSavingView(true); setError('')
    try {
      const saved = await api.saveView(view)
      setView(saved)
      setViews((current) => [...current.filter((item) => item.id !== saved.id), saved].sort((a, b) => a.id.localeCompare(b.id)))
      setDirty(false)
    } catch (current) { setError(errorText(current)) }
    finally { setSavingView(false); setBusy(false) }
  }
  const create = () => {
    if (!confirmDiscard()) return
    let index = views.length + 1
    while (views.some((item) => item.id === `view-${index}`)) index++
    setView(newView(index)); setDirty(true)
  }
  const remove = async (target: ViewDefinition) => {
    if (!window.confirm(`删除视图“${target.name}”？此操作不可恢复。`)) return
    if (target.revision === 0) { create(); return }
    try {
      await api.deleteView(target.id, target.revision)
      const next = views.filter((item) => item.id !== target.id)
      setViews(next); setView(next[0] || newView(1)); setDirty(false)
    } catch (current) { setError(errorText(current)) }
  }

  if (!settings) return <main className="boot-screen"><span className="brand-mark" aria-hidden="true">M</span><p>正在打开资源工作区…</p>{error && <p className="form-error" role="alert">{error}</p>}</main>
  if (showLogin || !activeProfile) return <LoginScreen settings={settings} busy={busy} error={error} onLogin={login} onSwitch={switchProfile} onDelete={deleteProfile} />

  return <DndContext onDragEnd={dragEnd}>
    <main className="app-shell">
      <a className="skip-link" href="#resource-workspace">跳到资源工作区</a>
      <header className="topbar">
        <h1 className="sr-only">MyFlowHub 资源工作区</h1>
        <div className="brand-lockup"><span className="brand-mark small" aria-hidden="true">M</span><div><strong>MyFlowHub</strong><small>Resource workspace</small></div></div>
        <button className="connection-pill" data-state={status.state} disabled={busy || status.state === 'connected' || status.state === 'connecting'} onClick={() => void connect()} aria-label={status.state === 'connected' ? '节点链路已连接' : '连接节点链路'} aria-live="polite">{status.state === 'connected' ? <Wifi aria-hidden="true" size={14} /> : <CloudOff aria-hidden="true" size={14} />}<span>{status.state}</span>{status.endpoint && <small>{status.endpoint}</small>}</button>
        <div className="topbar-actions">
          <Button variant="ghost" size="icon" aria-label="刷新节点树" onClick={() => void refreshPlatform(activeProfile)}><RefreshCw aria-hidden="true" size={16} /></Button>
          <Button variant="ghost" size="icon" aria-label="新建 Profile" onClick={() => setShowLogin(true)}><UserPlus aria-hidden="true" size={16} /></Button>
          <select aria-label="活动 Profile" name="active-profile" value={activeProfile.id} onChange={(event) => void switchProfile(event.target.value)}>
            {settings.profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name}</option>)}
          </select>
          <span className="profile-avatar small" aria-hidden="true">{activeProfile.name.slice(0, 1).toUpperCase()}</span>
        </div>
      </header>
      {error && <div className="error-banner" role="alert" aria-live="polite">{error}<button aria-label="关闭错误消息" onClick={() => setError('')}>×</button></div>}
      <div className="app-body">
        <aside className="sidebar">
          <Tabs defaultValue="explorer" className="sidebar-tabs">
            <TabsList><TabsTrigger value="explorer"><Boxes aria-hidden="true" size={14} />资源</TabsTrigger><TabsTrigger value="views"><Layers3 aria-hidden="true" size={14} />视图</TabsTrigger></TabsList>
            <TabsContent className="tabs-content" value="explorer"><Explorer topology={topology} resources={resources} selection={selection} onSelect={setSelection} onAdd={addResource} /></TabsContent>
            <TabsContent className="tabs-content" value="views"><ViewManager views={views} activeID={view.id} onOpen={(target) => { if (confirmDiscard()) { setView(target); setDirty(false) } }} onCreate={create} onDelete={(target) => void remove(target)} /></TabsContent>
          </Tabs>
          <footer className="sidebar-footer"><span className={`live-dot ${status.state === 'connected' ? 'live' : 'stopped'}`} aria-hidden="true" /><span><strong>Node {activeProfile.node_id}</strong><small>{topology.nodes.length} nodes · {resources.length} resources</small></span></footer>
        </aside>
        <Workspace id="resource-workspace" api={api} selection={selection} resources={resources} view={view} dirty={dirty} saving={savingView} onChange={changeView} onSave={() => void save()} />
      </div>
    </main>
  </DndContext>
}
