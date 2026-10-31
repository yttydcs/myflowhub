import * as Dialog from '@radix-ui/react-dialog'
import { Settings2, X } from 'lucide-react'
import type { DesktopAPI } from '../api'
import type { ResourceDescriptor } from '../types'
import { ResourceRendererSelector } from './Renderer'
import { Button } from './ui/button'

export function WidgetSettings({ api, resource, label, rendererID, onRendererChange }: {
  api: DesktopAPI
  resource: ResourceDescriptor
  label: string
  rendererID?: string
  onRendererChange(rendererID: string): void
}) {
  return <Dialog.Root>
    <Dialog.Trigger asChild>
      <button type="button" aria-label={`设置 ${label}`} title="组件设置"><Settings2 aria-hidden="true" size={14} /></button>
    </Dialog.Trigger>
    <Dialog.Portal>
      <Dialog.Overlay className="widget-settings-overlay" />
      <Dialog.Content className="widget-settings-dialog">
        <header className="widget-settings-header">
          <Dialog.Title>{label} 设置</Dialog.Title>
          <Dialog.Close asChild><Button variant="ghost" size="icon" aria-label="关闭组件设置"><X aria-hidden="true" size={16} /></Button></Dialog.Close>
        </header>
        <Dialog.Description className="widget-settings-description">调整当前组件的显示方式，更改会随视图保存。</Dialog.Description>
        <div className="widget-settings-field">
          <span>显示方式</span>
          <ResourceRendererSelector api={api} resource={resource} rendererID={rendererID} onRendererChange={onRendererChange} showDetails />
        </div>
        <dl className="inspector-facts">
          <div><dt>资源</dt><dd>{resource.id.name}</dd></div>
          <div><dt>所有者</dt><dd>{resource.id.owner_node_id}</dd></div>
        </dl>
        <footer className="widget-settings-footer"><Dialog.Close asChild><Button size="sm">完成</Button></Dialog.Close></footer>
      </Dialog.Content>
    </Dialog.Portal>
  </Dialog.Root>
}
