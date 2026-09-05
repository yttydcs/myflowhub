import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  DndContext,
  DragOverlay,
  closestCenter,
  pointerWithin,
  type Announcements,
  type CollisionDetection,
  type DragEndEvent,
  type DragMoveEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { Boxes, GripVertical, Layers3, Moon, Plus, RefreshCw, Settings2, Sun } from 'lucide-react'
import { api as productionApi, type DesktopAPI, type PreparedProfile } from './api'
import { useConnectionMonitor } from './discovery/useConnectionMonitor'
import { useDiscovery } from './discovery/useDiscovery'
import { CatalogContext } from './discovery/DiscoveryStatus'
import { BrandMark } from './components/BrandMark'
import { Explorer } from './components/Explorer'
import { Inspector } from './components/Inspector'
import { LoginScreen } from './components/LoginScreen'
import { Settings as DesktopSettings } from './components/Settings'
import { ViewManager, Workspace, type WorkspaceDockPreview } from './components/Workspace'
import { Button } from './components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './components/ui/tabs'
import { focusResourceAction, type FocusedResourceAction, type ResourceAction } from './lib/resource-actions'
import { errorText } from './lib/utils'
import { defaultUIPreferences, loadUIPreferences, saveUIPreferences, type Theme, type UIPreferences } from './preferences'
import { defaultRenderer, nextWidgetID } from './store'
import type { ConnectionStatus, Profile, ProfileState, ResourceDescriptor, Settings, ViewDefinition, ViewWidget, WorkspaceSelection } from './types'
import {
  addWorkspaceWidget,
  dockWorkspaceView,
  sameWorkspaceDockIntent,
  validateWorkspaceView,
  workspaceDropTargetPriority,
  workspacePanelDockSide,
  type WorkspaceDockIntent,
  type WorkspaceDockSide,
  type WorkspaceDockSource,
} from './workspace-layout'

type ActiveWorkspaceDrag = {
  source: WorkspaceDockSource
  widgetID: string
  label: string
}

type WorkspaceDropData = {
  kind?: 'workspace-divider' | 'workspace-root-edge' | 'workspace-panel' | 'workspace-background'
  widgetID?: string
  side?: WorkspaceDockSide
  parentPath?: number[]
  insertionIndex?: number
  beforeWidgetID?: string
  afterWidgetID?: string
}

type InspectorState = {
  selection: WorkspaceSelection
  focusedAction: FocusedResourceAction | null
}

const workspaceCollisionDetection: CollisionDetection = (args) => {
  const pointerCollisions = pointerWithin(args)
  if (pointerCollisions.length === 0) return args.pointerCoordinates ? [] : closestCenter(args)
  return [...pointerCollisions].sort((left, right) => workspaceDropTargetPriority(String(left.id)) - workspaceDropTargetPriority(String(right.id)))
}

function dragDataLabel(data: Record<string, unknown> | undefined): string {
  if (typeof data?.label === 'string') return data.label
  const resource = data?.resource as ResourceDescriptor | undefined
  if (resource) return resource.presentation?.label || resource.id.name
  return '面板'
}

function dropDataLabel(data: Record<string, unknown> | undefined): string {
  const candidate = data as WorkspaceDropData | undefined
  if (candidate?.kind === 'workspace-divider') return `分隔线 ${candidate.beforeWidgetID} 与 ${candidate.afterWidgetID} 之间`
  if (candidate?.kind === 'workspace-root-edge' && candidate.side) return `工作区${dockSideLabel(candidate.side)}`
  if (candidate?.kind === 'workspace-panel' && candidate.widgetID) return `面板 ${candidate.widgetID}；移动到边缘选择停靠方向`
  if (candidate?.kind === 'workspace-background') return '空工作区'
  return '当前区域'
}

const workspaceAnnouncements: Announcements = {
  onDragStart: ({ active }) => `已抓取 ${dragDataLabel(active.data.current)}。使用方向键移动，空格放置，Escape 取消。`,
  onDragOver: ({ over }) => over ? `当前位于${dropDataLabel(over.data.current)}。` : '当前不在有效停靠区域。',
  onDragEnd: ({ active, over }) => over
    ? `已在${dropDataLabel(over.data.current)}释放 ${dragDataLabel(active.data.current)}。`
    : `未移动 ${dragDataLabel(active.data.current)}。`,
  onDragCancel: ({ active }) => `已取消移动 ${dragDataLabel(active.data.current)}。`,
}

function dragPoint(event: DragEndEvent | DragMoveEvent): { x: number; y: number } | undefined {
  const translated = event.active.rect.current.translated
  if (!translated) return undefined
  const activator = event.activatorEvent
  if ('clientX' in activator && 'clientY' in activator
    && typeof activator.clientX === 'number' && typeof activator.clientY === 'number') {
    return { x: activator.clientX + event.delta.x, y: activator.clientY + event.delta.y }
  }
  return { x: translated.left + translated.width / 2, y: translated.top + translated.height / 2 }
}

function panelDockSide(event: DragEndEvent | DragMoveEvent): WorkspaceDockSide | undefined {
  const over = event.over
  const point = dragPoint(event)
  if (!over || !point) return undefined
  const x = (point.x - over.rect.left) / Math.max(over.rect.width, 1)
  const y = (point.y - over.rect.top) / Math.max(over.rect.height, 1)
  return workspacePanelDockSide(x, y)
}

function resourceWidget(resource: ResourceDescriptor, widgets: ViewWidget[]): ViewWidget {
  return {
    id: nextWidgetID(resource, widgets),
    owner_node_id: resource.id.owner_node_id,
    resource_name: resource.id.name,
    renderer: defaultRenderer(resource),
  }
}

function dockSideLabel(value: WorkspaceDockSide): string {
  return ({ left: '左侧', right: '右侧', top: '上方', bottom: '下方' })[value]
}

function dockDescription(intent: WorkspaceDockIntent): string {
  if (intent.kind === 'panel-edge') return `停靠预览：置于 ${intent.targetWidgetID} ${dockSideLabel(intent.side)}`
  if (intent.kind === 'root-edge') return `停靠预览：置于整个工作区${dockSideLabel(intent.side)}`
  if (intent.kind === 'split-gap') return `停靠预览：插入 ${intent.beforeWidgetID} 与 ${intent.afterWidgetID} 之间`
  return '停靠预览：填满空工作区'
}

function newView(index = 1): ViewDefinition {
  return { id: `view-${index}`, name: `工作视图 ${index}`, revision: 0, widgets: [] }
}

export function App({ api = productionApi }: { api?: DesktopAPI }) {
  const [settings, setSettings] = useState<Settings>()
  const [profileStates, setProfileStates] = useState<ProfileState[]>([])
  const [status, setStatus] = useState<ConnectionStatus>({ state: 'signed_out' })
  const { controller: discovery, snapshot: discoveryState } = useDiscovery(api)
  const { topology, resources } = discoveryState
  const connectionGeneration = useRef(0)
  const statusRequest = useRef(0)
  const profileGeneration = useRef(0)
  const platformChecks = useRef(0)
  const changingConnection = useRef(false)
  const contextProfile = useRef('')
  const connectionSignature = useRef('')
  const browseRoot = useRef({ profile: '', root: '' })
  const [catalogOwner, setCatalogOwner] = useState('')
  const invalidateDiscovery = useCallback(() => {
    changingConnection.current = true
    connectionGeneration.current += 1
    statusRequest.current += 1
    contextProfile.current = ''
    connectionSignature.current = ''
    discovery.reset()
    setCatalogOwner('')
  }, [discovery])
  const [inspector, setInspector] = useState<InspectorState>({ selection: null, focusedAction: null })
  const { selection, focusedAction: focusedResourceAction } = inspector
  const [views, setViews] = useState<ViewDefinition[]>([])
  const [view, setView] = useState<ViewDefinition>(newView())
  const [viewProfile, setViewProfile] = useState<{ id: string; generation: number }>()
  const viewEditGeneration = useRef(0)
  const viewLoadRequest = useRef(0)
  const currentView = useRef(view)
  currentView.current = view
  const [activeContent, setActiveContent] = useState<'workspace' | 'settings'>('workspace')
  const [preferences, setPreferences] = useState<UIPreferences>(defaultUIPreferences())
  const [dirty, setDirty] = useState(false)
  const [busy, setBusy] = useState(false)
  const [savingView, setSavingView] = useState(false)
  const [activeWorkspaceDrag, setActiveWorkspaceDrag] = useState<ActiveWorkspaceDrag | null>(null)
  const [dockIntent, setDockIntent] = useState<WorkspaceDockIntent | null>(null)
  const [error, setError] = useState('')

  useEffect(() => { if (!busy) changingConnection.current = false }, [busy])

  const activeProfile = useMemo(
    () => settings?.profiles.find((profile) => profile.id === settings.active_profile_id),
    [settings],
  )
  const viewReady = viewProfile?.id === activeProfile?.id && viewProfile?.generation === profileGeneration.current
  const invalidateProfile = () => {
    profileGeneration.current += 1
    setViewProfile(undefined)
    setInspector({ selection: null, focusedAction: null })
    invalidateDiscovery()
  }
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

  useEffect(() => {
    if (activeContent !== 'workspace' || !selection || selection.kind !== 'resource') {
      setInspector((current) => current.focusedAction ? { ...current, focusedAction: null } : current)
    }
  }, [activeContent, selection])

  const loadViews = useCallback(async (profileID: string) => {
    const request = ++viewLoadRequest.current
    const generation = profileGeneration.current
    const document = await api.views()
    if (generation !== profileGeneration.current || request !== viewLoadRequest.current) return
    if (document.version !== 3) throw new Error(`不支持的视图文档版本：${document.version}`)
    document.views.forEach(validateWorkspaceView)
    const activeView = document.views[0] || newView(1)
    setViews(document.views)
    viewEditGeneration.current += 1
    setView(activeView)
    setViewProfile({ id: profileID, generation })
    setDirty(false)
  }, [api])

  const refreshEntryState = useCallback(async () => {
    const generation = profileGeneration.current
    const [nextSettings, nextProfileStates] = await Promise.all([api.settings(), api.profileStates()])
    if (generation !== profileGeneration.current) return nextSettings
    setSettings(nextSettings)
    setProfileStates(nextProfileStates)
    return nextSettings
  }, [api])

  const acceptStatus = useCallback((profile: Profile, nextStatus: ConnectionStatus) => {
    const signature = JSON.stringify([profile.id, nextStatus.state, nextStatus.generation, nextStatus.link_generation, nextStatus.parent_node_id, nextStatus.endpoint])
    const changed = signature !== connectionSignature.current
    if (changed) {
      connectionGeneration.current += 1
      discovery.reset()
      connectionSignature.current = signature
    }
    setStatus(nextStatus)
    if (nextStatus.state !== 'connected') return false
    const root = browseRoot.current.profile === profile.id ? browseRoot.current.root : nextStatus.parent_node_id || profile.parent_node_id
    contextProfile.current = profile.id
    discovery.activate(profile.id, connectionGeneration.current, root)
    return changed
  }, [discovery])

  const refreshPlatform = useCallback(async (profile: Profile, waitForAutoConnect = true, loadTree = true) => {
    platformChecks.current += 1
    try {
      const request = ++statusRequest.current
      setError('')
      let nextStatus = await api.status()
      for (let attempt = 0; request === statusRequest.current && waitForAutoConnect && profile.auto_connect && attempt < 60 && (nextStatus.state === 'disconnected' || nextStatus.state === 'connecting'); attempt += 1) {
        await new Promise((resolve) => window.setTimeout(resolve, 250))
        if (request !== statusRequest.current) return
        nextStatus = await api.status()
      }
      if (request !== statusRequest.current) return
      acceptStatus(profile, nextStatus)
      if (nextStatus.state !== 'connected') return
      if (loadTree) void discovery.query(discovery.getSnapshot().scope)
    } finally { platformChecks.current -= 1 }
  }, [api, discovery, acceptStatus])

  const checkConnection = useCallback(async (current: () => boolean) => {
    if (!activeProfile || changingConnection.current || platformChecks.current > 0) return
    const generation = connectionGeneration.current
    const request = ++statusRequest.current
    try {
      const next = await api.status()
      if (!current() || request !== statusRequest.current || generation !== connectionGeneration.current) return
      if (acceptStatus(activeProfile, next) && next.state === 'connected') void discovery.query(discovery.getSnapshot().scope)
    } catch (error) {
      if (current() && request === statusRequest.current && generation === connectionGeneration.current) {
        acceptStatus(activeProfile, { state: 'failed', last_error: errorText(error) })
        setError(`读取本机连接状态失败：${errorText(error)}`)
      }
    }
  }, [activeProfile, api, acceptStatus, discovery])
  useConnectionMonitor(!!activeProfile, checkConnection)

  useEffect(() => {
    if (!discoveryState.scope || contextProfile.current !== activeProfile?.id) return
    const expanded = preferences.expanded_node_ids ?? [discoveryState.scope]
    discovery.restoreExpanded(expanded, preferences.focused_node_id)
  }, [activeProfile?.id, discovery, discoveryState.topology, discoveryState.scope, preferences.expanded_node_ids, preferences.focused_node_id])

  useEffect(() => {
    if (!discoveryState.scope || contextProfile.current !== activeProfile?.id) return
    const selectedOwner = selection?.kind === 'resource' ? selection.resource.id.owner_node_id : selection?.kind === 'node' ? selection.node.node_id : catalogOwner || discoveryState.scope
    const owners = new Set([selectedOwner, ...(viewReady ? view.widgets.map((widget) => widget.owner_node_id) : [])])
    discovery.setCatalogOwners(owners)
    for (const owner of owners) void discovery.catalog(owner)
  }, [activeProfile?.id, discovery, discoveryState.scope, discoveryState.session, catalogOwner, selection, view.widgets, viewReady])

  useEffect(() => {
    setInspector((current) => {
      const selected = current.selection
      if (selected?.kind === 'node') {
        const node = topology.nodes.find((item) => item.node_id === selected.node.node_id)
        if (!node) return { selection: null, focusedAction: null }
        if (node !== selected.node) return { ...current, selection: { kind: 'node', node } }
      }
      if (selected?.kind === 'resource') {
        const catalog = discoveryState.catalogs.get(selected.resource.id.owner_node_id)
        if (catalog?.status !== 'loaded') return current
        const resource = catalog.catalog?.resources.find((item) => item.id.name === selected.resource.id.name)
        if (!resource) return { selection: null, focusedAction: null }
        if (resource !== selected.resource) return { ...current, selection: { kind: 'resource', resource } }
      }
      return current
    })
  }, [topology, discoveryState.catalogs])

  useEffect(() => {
    let active = true
    const generation = profileGeneration.current
    void (async () => {
      try {
        const nextSettings = await refreshEntryState()
        if (!active || generation !== profileGeneration.current) return
        const profile = nextSettings.profiles.find((item) => item.id === nextSettings.active_profile_id)
        if (profile) await Promise.all([refreshPlatform(profile), loadViews(profile.id)])
      } catch (current) {
        if (active && generation === profileGeneration.current) setError(errorText(current))
      }
    })()
    return () => { active = false; statusRequest.current += 1 }
  }, [loadViews, refreshEntryState, refreshPlatform])

  function updatePreferences(patch: Partial<Omit<UIPreferences, 'version'>>) {
    const next: UIPreferences = { ...preferences, ...patch, version: 1 }
    setPreferences(next)
    if (!activeProfile) return
    const saveError = saveUIPreferences(activeProfile.id, next)
    if (saveError) setError(saveError)
  }

  const login = async (profile: Profile, permit: string, allowTOFU: boolean) => {
    if (!confirmDiscard()) return false
    setBusy(true)
    setError('')
    try {
      invalidateProfile()
      const saved = await api.login(profile, permit, allowTOFU)
      const next = await refreshEntryState()
      const nextProfile = next.profiles.find((item) => item.id === next.active_profile_id) || saved
      await Promise.all([refreshPlatform(nextProfile), loadViews(nextProfile.id)])
      setActiveContent('workspace')
      setInspector({ selection: null, focusedAction: null })
      return true
    } catch (current) {
      const operationError = errorText(current)
      try {
        await refreshEntryState()
        setError(operationError)
      } catch (refreshError) {
        setError(`${operationError}；刷新 Profile 状态失败：${errorText(refreshError)}`)
      }
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
      await refreshEntryState()
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
      invalidateProfile()
      const saved = await api.saveProfile(profile)
      await refreshEntryState()
      setInspector({ selection: null, focusedAction: null })
      await Promise.all([refreshPlatform(saved, false), loadViews(saved.id)])
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
      invalidateProfile()
      await api.switchProfile(profileID)
      const next = await refreshEntryState()
      const profile = next.profiles.find((item) => item.id === profileID)
      if (profile) await Promise.all([refreshPlatform(profile), loadViews(profile.id)])
      setInspector({ selection: null, focusedAction: null })
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
      if (profileID === activeProfile?.id) invalidateProfile()
      await api.deleteProfile(profileID, `DELETE ${profileID}`)
      await refreshEntryState()
      if (profileID === activeProfile?.id) {
        setStatus({ state: 'signed_out' })
        invalidateDiscovery()
        setViews([])
        setView(newView())
        setDirty(false)
      }
      setInspector({ selection: null, focusedAction: null })
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
      invalidateDiscovery()
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
      invalidateDiscovery()
      await api.disconnect()
      setStatus(await api.status())
      invalidateDiscovery()
      setInspector({ selection: null, focusedAction: null })
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const deactivateProfile = async () => {
    if (!confirmDiscard()) return
    setBusy(true)
    setError('')
    let operationError = ''
    try {
      invalidateProfile()
      await api.deactivateProfile()
    } catch (current) {
      operationError = errorText(current)
    }
    try {
      const next = await refreshEntryState()
      if (!next.active_profile_id) {
        setStatus({ state: 'signed_out' })
        invalidateDiscovery()
        setInspector({ selection: null, focusedAction: null })
        setViews([])
        setView(newView())
        setDirty(false)
        setActiveContent('workspace')
      }
      if (operationError) setError(operationError)
    } catch (current) {
      setError(operationError ? `${operationError}；刷新 Profile 状态失败：${errorText(current)}` : errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const refresh = async () => {
    if (!activeProfile) return
    setBusy(true)
    try {
      await refreshPlatform(activeProfile, true, false)
      await discovery.refresh(preferences.expanded_node_ids ?? [discovery.getSnapshot().scope], preferences.focused_node_id)
    } catch (current) {
      setError(errorText(current))
    } finally {
      setBusy(false)
    }
  }

  const changeView = (next: ViewDefinition) => {
    try {
      if (!viewReady) return
      validateWorkspaceView(next)
      viewEditGeneration.current += 1
      setView(next)
      setDirty(true)
    } catch (current) {
      setError(errorText(current))
    }
  }
  const addResource = (resource: ResourceDescriptor) => {
    setActiveContent('workspace')
    try {
      changeView(addWorkspaceWidget(view, resourceWidget(resource, view.widgets)))
    } catch (current) {
      setError(errorText(current))
    }
  }
  const selectWorkspaceItem = (next: WorkspaceSelection) => {
    setInspector({ selection: next, focusedAction: null })
  }
  const focusResourceCapability = (resource: ResourceDescriptor, action: ResourceAction) => {
    setActiveContent('workspace')
    setInspector({
      selection: { kind: 'resource', resource },
      focusedAction: focusResourceAction(resource, action.capability),
    })
  }
  const closeInspector = () => {
    setInspector({ selection: null, focusedAction: null })
  }
  const clearFocusedResourceAction = () => {
    setInspector((current) => current.focusedAction ? { ...current, focusedAction: null } : current)
  }
  const activeDrag = (event: DragStartEvent | DragMoveEvent | DragEndEvent): ActiveWorkspaceDrag | null => {
    const data = event.active.data.current as {
      kind?: string
      widgetID?: string
      label?: string
      resource?: ResourceDescriptor
    } | undefined
    if (data?.kind === 'workspace-widget' && data.widgetID) {
      const widget = view.widgets.find((candidate) => candidate.id === data.widgetID)
      return widget ? { source: { kind: 'existing', widgetID: widget.id }, widgetID: widget.id, label: data.label || widget.resource_name } : null
    }
    if (data?.kind === 'resource' && data.resource) {
      const widget = resourceWidget(data.resource, view.widgets)
      return {
        source: { kind: 'new', widget },
        widgetID: widget.id,
        label: data.resource.presentation?.label || data.resource.id.name,
      }
    }
    return null
  }
  const resolveDockIntent = (event: DragMoveEvent | DragEndEvent): WorkspaceDockIntent | null => {
    const over = event.over
    if (!over) return null
    const data = over.data.current as WorkspaceDropData | undefined
    if (data?.kind === 'workspace-divider'
      && Array.isArray(data.parentPath)
      && typeof data.insertionIndex === 'number'
      && data.beforeWidgetID
      && data.afterWidgetID) {
      return {
        kind: 'split-gap',
        parentPath: data.parentPath,
        insertionIndex: data.insertionIndex,
        beforeWidgetID: data.beforeWidgetID,
        afterWidgetID: data.afterWidgetID,
      }
    }
    if (data?.kind === 'workspace-root-edge' && data.side) return { kind: 'root-edge', side: data.side }
    if (data?.kind === 'workspace-panel' && data.widgetID) {
      const side = panelDockSide(event)
      return side ? { kind: 'panel-edge', targetWidgetID: data.widgetID, side } : null
    }
    if (data?.kind === 'workspace-background') {
      return view.layout_root ? null : { kind: 'empty-workspace' }
    }
    return null
  }
  const dragStart = (event: DragStartEvent) => {
    setActiveWorkspaceDrag(activeDrag(event))
    setDockIntent(null)
  }
  const dragMove = (event: DragMoveEvent) => {
    const next = resolveDockIntent(event)
    setDockIntent((current) => sameWorkspaceDockIntent(current, next) ? current : next)
  }
  const dragEnd = (event: DragEndEvent) => {
    const currentDrag = activeWorkspaceDrag || activeDrag(event)
    const intent = resolveDockIntent(event) || dockIntent
    setActiveWorkspaceDrag(null)
    setDockIntent(null)
    if (!currentDrag || !intent) return
    try {
      const next = dockWorkspaceView(view, currentDrag.source, intent)
      if (next !== view) {
        setActiveContent('workspace')
        changeView(next)
      }
    } catch (current) {
      setError(errorText(current))
    }
  }
  const dragCancel = () => {
    setActiveWorkspaceDrag(null)
    setDockIntent(null)
  }
  const dockPreview = useMemo<WorkspaceDockPreview | null>(() => {
    if (!activeWorkspaceDrag || !dockIntent) return null
    try {
      const next = dockWorkspaceView(view, activeWorkspaceDrag.source, dockIntent)
      if (next === view) return null
      return { view: next, widgetID: activeWorkspaceDrag.widgetID, description: dockDescription(dockIntent) }
    } catch {
      return null
    }
  }, [activeWorkspaceDrag, dockIntent, view])
  const saveView = async () => {
    if (!viewReady || savingView) return
    const generation = profileGeneration.current
    const editGeneration = viewEditGeneration.current
    const submitted = view
    setBusy(true)
    setSavingView(true)
    setError('')
    try {
      const saved = await api.saveView(submitted)
      if (generation !== profileGeneration.current) return
      validateWorkspaceView(saved)
      if (saved.id !== submitted.id) throw new Error('保存结果引用了其他 View')
      setViews((current) => [...current.filter((item) => item.id !== saved.id), saved].sort((left, right) => left.id.localeCompare(right.id)))
      if (currentView.current.id === submitted.id) {
        if (editGeneration === viewEditGeneration.current) {
          setView(saved)
          setDirty(false)
        } else {
          // Keep edits made during the save, but advance the optimistic revision
          // so their next explicit save can succeed against the acknowledged version.
          setView((current) => current.id === submitted.id ? { ...current, revision: saved.revision } : current)
        }
      }
    } catch (current) {
      if (generation === profileGeneration.current) setError(errorText(current))
    } finally {
      setSavingView(false)
      if (generation === profileGeneration.current) setBusy(false)
    }
  }

  const createView = () => {
    if (!viewReady) return
    if (!confirmDiscard()) return
    let index = views.length + 1
    while (views.some((item) => item.id === `view-${index}`)) index += 1
    viewEditGeneration.current += 1
    setView(newView(index))
    setDirty(true)
    setActiveContent('workspace')
  }
  const removeView = async (target: ViewDefinition) => {
    if (!window.confirm(`删除视图“${target.name}”？此操作不可恢复。`)) return
    if (target.revision === 0) {
      viewEditGeneration.current += 1
      setView(views[0] || newView())
      setDirty(false)
      return
    }
    try {
      await api.deleteView(target.id, target.revision)
      const next = views.filter((item) => item.id !== target.id)
      setViews(next)
      viewEditGeneration.current += 1
      setView(next[0] || newView(1))
      setDirty(false)
    } catch (current) {
      setError(errorText(current))
    }
  }

  if (!settings) {
    return <main className="boot-screen"><BrandMark /><p>正在打开资源工作区…</p>{error && <p className="form-error" role="alert">{error}</p>}</main>
  }
  if (!activeProfile) {
    return (
      <LoginScreen
        settings={settings}
        profileStates={profileStates}
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
    <CatalogContext.Provider value={{ states: discoveryState.catalogs, retry: (owner) => void discovery.catalog(owner, true) }}>
    <DndContext
      accessibility={{
        announcements: workspaceAnnouncements,
        screenReaderInstructions: { draggable: '按空格抓取面板或资源，使用方向键移动，按空格放置，按 Escape 取消。' },
      }}
      collisionDetection={workspaceCollisionDetection}
      onDragStart={dragStart}
      onDragMove={dragMove}
      onDragEnd={dragEnd}
      onDragCancel={dragCancel}
    >
      <main className="app-shell">
        <a className="skip-link" href="#resource-workspace">跳到资源工作区</a>
        <header className="topbar">
          <h1 className="sr-only">MyFlowHub 资源工作区</h1>
          <div className="brand-lockup"><BrandMark size="compact" /><strong>MyFlowHub</strong></div>
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
                  key={activeProfile.id}
                  topology={topology}
                  discovery={discoveryState}
                  scopeRoot={discoveryState.scope || status.parent_node_id || activeProfile.parent_node_id}
                  defaultRoot={status.parent_node_id || activeProfile.parent_node_id}
                  onScopeRootChange={(root, followParent) => {
                    try {
                      discovery.activate(activeProfile.id, connectionGeneration.current, root)
                      browseRoot.current = followParent ? { profile: '', root: '' } : { profile: activeProfile.id, root }
                      updatePreferences({ focused_node_id: undefined, expanded_node_ids: [...new Set([...(preferences.expanded_node_ids || []), root])] })
                      setInspector({ selection: null, focusedAction: null })
                      setCatalogOwner(root)
                      void discovery.query(root)
                    } catch (current) { setError(errorText(current)) }
                  }}
                  onLoadSubtree={() => void discovery.query(discoveryState.scope, 0, true)}
                  onExpandNode={(owner) => void discovery.query(owner)}
                  onRetryNode={(owner) => void discovery.query(owner, 1, true)}
                  onCatalogOwnerChange={setCatalogOwner}
                  onRetryCatalog={(owner) => void discovery.catalog(owner, true)}
                  resources={resources}
                  selection={selection}
                  expandedNodeIDs={preferences.expanded_node_ids}
                  expandedResourcePaths={preferences.expanded_resource_paths}
                  focusedNodeID={preferences.focused_node_id}
                  splitRatio={preferences.explorer_split_ratio}
                  collapsedPane={preferences.collapsed_explorer_pane}
                  onExpandedNodeIDsChange={(expandedNodeIDs) => updatePreferences({ expanded_node_ids: expandedNodeIDs })}
                  onExpandedResourcePathsChange={(expandedResourcePaths) => updatePreferences({ expanded_resource_paths: expandedResourcePaths })}
                  onFocusedNodeIDChange={(focusedNodeID) => updatePreferences({ focused_node_id: focusedNodeID })}
                  onSplitRatioChange={(splitRatio) => updatePreferences({ explorer_split_ratio: splitRatio })}
                  onCollapsedPaneChange={(collapsedPane) => updatePreferences({ collapsed_explorer_pane: collapsedPane })}
                  onSelect={selectWorkspaceItem}
                  onAdd={addResource}
                  onAction={focusResourceCapability}
                />
              </TabsContent>
              <TabsContent className="tabs-content" value="views">
                <ViewManager
                  views={viewReady ? views : []}
                  activeID={view.id}
                  onOpen={(target) => {
                    if (!viewReady || !confirmDiscard()) return
                    viewEditGeneration.current += 1
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
              <button role="tab" aria-selected={activeContent === 'workspace'} className={activeContent === 'workspace' ? 'is-active' : ''} onClick={() => setActiveContent('workspace')}><Layers3 aria-hidden="true" size={13} />{viewReady ? view.name : '视图加载中…'}{viewReady && dirty && <span className="tab-dirty" aria-label="未保存">●</span>}</button>
              <button role="tab" aria-selected={activeContent === 'settings'} className={activeContent === 'settings' ? 'is-active' : ''} onClick={() => setActiveContent('settings')}><Settings2 aria-hidden="true" size={13} />设置</button>
              <button className="new-tab" disabled={!viewReady} onClick={createView} aria-label="新建视图"><Plus aria-hidden="true" size={14} /></button>
            </nav>
            {activeContent === 'settings'
              ? (
                <DesktopSettings
                  api={api}
                  settings={settings}
                  activeProfile={activeProfile}
                  status={status}
                  theme={preferences.theme}
                  busy={busy}
                  onConnect={connect}
                  onDisconnect={disconnect}
                  onDeactivateProfile={deactivateProfile}
                  onSwitchProfile={switchProfile}
                  onSaveProfile={saveProfile}
                  onDeleteProfile={deleteProfile}
                  onThemeChange={(theme: Theme) => updatePreferences({ theme })}
                />
              )
              : !viewReady ? <section className="workspace" aria-label="资源工作区"><p className="discovery-status" role="status">正在加载当前 Profile 的视图…</p><button type="button" onClick={() => void loadViews(activeProfile.id).catch((current) => setError(errorText(current)))}>重新加载视图</button></section> : (
                <Workspace
                  key={activeProfile.id}
                  id="resource-workspace"
                  api={api}
                  resources={resources}
                  view={view}
                  dirty={dirty}
                  saving={savingView}
                  dockPreview={dockPreview}
                  onChange={changeView}
                  onSave={() => void saveView()}
                  onError={setError}
                />
              )}
          </div>

          {inspectorVisible && (
            <Inspector
              api={api}
              selection={selection}
              resources={resources}
              focusedAction={focusedResourceAction}
              onClose={closeInspector}
              onClearAction={clearFocusedResourceAction}
              onAdd={addResource}
              onFocusNode={(nodeID) => updatePreferences({
                focused_node_id: nodeID,
                expanded_node_ids: [...new Set([...(preferences.expanded_node_ids || []), nodeID])],
              })}
            />
          )}
        </div>
      </main>
      <DragOverlay dropAnimation={null}>
        {activeWorkspaceDrag && (
          <div className="workspace-drag-overlay" aria-hidden="true">
            <GripVertical size={14} />
            <span>{activeWorkspaceDrag.label}</span>
          </div>
        )}
      </DragOverlay>
    </DndContext>
    </CatalogContext.Provider>
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
