import { createContext, useContext, useLayoutEffect, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode } from 'react'
import { MoveDiagonal2 } from 'lucide-react'
import type { ViewWidget } from '../types'

type Size = NonNullable<ViewWidget['compact_size']>
export const CompactLayoutContext = createContext<{
  columns: number
  columnWidth: number
  order: string[]
  insert?: { targetID: string; before: boolean }
  onResize(id: string, size?: Size): void
} | undefined>(undefined)

const clamp = (value: number, minimum: number, maximum: number) => Math.max(minimum, Math.min(maximum, value))

// Keep the same leaf and renderer mounted in both modes, including invalid
// local form input and in-flight writes. Only the layout CSS changes.
export function CompactCard({ widget, children }: { widget: ViewWidget; children: ReactNode }) {
  const layout = useContext(CompactLayoutContext)
  const root = useRef<HTMLDivElement>(null)
  const drag = useRef<{ x: number; y: number; width: number; height: number; size: Size }>()
  const [preview, setPreview] = useState<Size>()
  const [rows, setRows] = useState(8)
  const size = preview || widget.compact_size
  useLayoutEffect(() => {
    if (!layout || !root.current) return
    const card = root.current.querySelector('article')
    if (!card) return
    const measure = () => setRows(Math.max(1, Math.ceil((card.getBoundingClientRect().height + 12) / 16)))
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(card)
    return () => observer.disconnect()
  }, [!!layout])

  const finish = (event: PointerEvent<HTMLButtonElement>, commit: boolean) => {
    const current = drag.current
    drag.current = undefined
    setPreview(undefined)
    if (!current) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    if (commit) layout?.onResize(widget.id, current.size)
  }
  const insertion = layout?.insert?.targetID === widget.id ? (layout.insert.before ? ' insert-before' : ' insert-after') : ''
  const style: CSSProperties | undefined = layout ? {
    order: layout.order.indexOf(widget.id),
    gridColumn: `span ${Math.min(size?.columns || 1, layout.columns)}`,
    gridRow: `span ${rows}`,
    '--compact-card-height': size?.height ? `${size.height}px` : 'auto',
  } as CSSProperties : undefined
  return <div ref={root} className={`workspace-layout-leaf${layout && size?.height ? ' is-sized' : ''}${insertion}`} style={style}>
    {children}
    {layout && <button className="compact-card-resize" aria-label={`调整 ${widget.resource_name} 卡片尺寸`} title="拖动调整大小；方向键调整，Home 或双击恢复自动大小"
      onPointerDown={(event) => {
        if (event.button !== 0) return
        const bounds = root.current?.querySelector('article')?.getBoundingClientRect()
        if (!bounds) return
        event.preventDefault(); event.stopPropagation(); event.currentTarget.focus()
        event.currentTarget.setPointerCapture(event.pointerId)
        drag.current = { x: event.clientX, y: event.clientY, width: bounds.width, height: bounds.height, size: { columns: Math.min(size?.columns || 1, layout.columns), height: Math.round(bounds.height) } }
      }}
      onPointerMove={(event) => {
        const current = drag.current
        if (!current) return
        current.size = {
          columns: clamp(Math.round((current.width + event.clientX - current.x + 12) / (layout.columnWidth + 12)), 1, layout.columns),
          height: clamp(Math.round(current.height + event.clientY - current.y), 104, 1200),
        }
        setPreview(current.size)
      }}
      onPointerUp={(event) => finish(event, true)} onPointerCancel={(event) => finish(event, false)} onLostPointerCapture={(event) => finish(event, false)}
      onDoubleClick={() => layout.onResize(widget.id)}
      onKeyDown={(event) => {
        if (event.key === 'Escape' && drag.current) { drag.current = undefined; setPreview(undefined); event.preventDefault(); return }
        if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home'].includes(event.key)) return
        event.preventDefault(); event.stopPropagation()
        if (event.key === 'Home') { layout.onResize(widget.id); return }
        const height = size?.height || Math.round(root.current?.querySelector('article')?.getBoundingClientRect().height || 104)
        const columns = Math.min(size?.columns || 1, layout.columns)
        layout.onResize(widget.id, event.key === 'ArrowLeft' || event.key === 'ArrowRight'
          ? { ...size, columns: clamp(columns + (event.key === 'ArrowRight' ? 1 : -1), 1, layout.columns) }
          : { columns, height: clamp(height + (event.key === 'ArrowDown' ? 24 : -24), 104, 1200) })
      }}><MoveDiagonal2 size={12} aria-hidden="true" /></button>}
  </div>
}
