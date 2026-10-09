import { useId, useState, type ReactNode } from 'react'
import { ChevronRight } from 'lucide-react'
import type { DataSchemaDefinition } from '../generated/variable-contract.generated'
import { Button } from '../components/ui/button'
import { Progress } from '../components/ui/progress'

const isStructured = (value: unknown) => value !== null && typeof value === 'object'
const sizeLabel = (value: unknown) => Array.isArray(value) ? `${value.length} 项` : `${Object.keys(value as object).length} 个字段`

function ValueGroup({ label, value, initiallyOpen = false, children }: { label: string; value: unknown; initiallyOpen?: boolean; children: ReactNode }) {
  const [open, setOpen] = useState(initiallyOpen)
  const id = useId()
  return <div className="variable-value-group">
    <Button variant="ghost" className="variable-value-toggle" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}>
      <ChevronRight size={14} aria-hidden="true" className={open ? 'is-open' : ''} />
      <span className="variable-value-label">{label}</span><span className="variable-value-count">{sizeLabel(value)}</span>
    </Button>
    <div id={id} hidden={!open} className="variable-value-children">{open && children}</div>
  </div>
}

export function VariableValue({ value, definition, depth = 0, rendererID = '' }: { value: unknown; definition: DataSchemaDefinition; depth?: number; rendererID?: string }) {
  if (definition.write_only) return <span>此字段不回显</span>
  if (value === null) return <span className="variable-empty-value">null</span>
  if (typeof value === 'boolean') return rendererID.includes('text') ? <span>{String(value)}</span> : <div className={`boolean-value ${value ? 'is-true' : 'is-false'}`}><strong>{value ? '开启' : '关闭'}</strong><span className="sr-only">{String(value)}</span></div>
  if (typeof value === 'number') return <div className="numeric-value"><strong>{definition.precision === undefined ? String(value) : value.toFixed(definition.precision)}{definition.unit ? ` ${definition.unit}` : ''}</strong>{rendererID.includes('progress') && definition.minimum !== undefined && definition.maximum !== undefined && definition.maximum > definition.minimum && <Progress aria-label="当前确认值" max={definition.maximum - definition.minimum} value={value - definition.minimum} />}</div>
  if (typeof value === 'string') return rendererID.includes('code') ? <pre>{value === '' ? '""' : value}</pre> : <span className={`multiline-value${value === '' ? ' variable-empty-value' : ''}`}>{value === '' ? '""' : value}</span>
  if (isStructured(value) && depth >= 8) return <span className="variable-empty-value">{sizeLabel(value)}；已达展示深度上限</span>
  if (Array.isArray(value)) return <div className="variable-value-array">
    {value.length === 0 && <span className="variable-empty-value">空数组</span>}
    <ol>{value.slice(0, 100).map((item, index) => <li key={index}>
      {isStructured(item) && !definition.items!.write_only
        ? <ValueGroup label={`第 ${index + 1} 项`} value={item}><VariableValue value={item} definition={definition.items!} depth={depth + 1} /></ValueGroup>
        : <div className="variable-value-item"><span className="variable-value-count">{index + 1}</span><VariableValue value={item} definition={definition.items!} depth={depth + 1} /></div>}
    </li>)}</ol>
    {value.length > 100 && <p className="variable-empty-value">仅显示前 100 项，共 {value.length} 项</p>}
  </div>
  if (value && typeof value === 'object') {
    const properties = new Map(definition.properties?.map((property) => [property.name, property.schema]))
    const entries = Object.entries(value)
    return <div className="variable-value-object">
      {entries.length === 0 && <span className="variable-empty-value">空对象</span>}
      <dl>{entries.slice(0, 128).map(([name, item]) => {
        const child = properties.get(name) || definition.additional_properties!
        const label = child.title || name
        const structured = isStructured(item) && !child.write_only
        return <div key={name} className={structured ? 'is-structured' : 'variable-value-field'}>
          <dt className={structured ? 'sr-only' : undefined}>{label}</dt>
          <dd>{structured
            ? <ValueGroup label={label} value={item} initiallyOpen={depth === 0}><VariableValue value={item} definition={child} depth={depth + 1} /></ValueGroup>
            : <VariableValue value={item} definition={child} depth={depth + 1} />}</dd>
        </div>
      })}</dl>
      {entries.length > 128 && <p className="variable-empty-value">仅显示前 128 个字段，共 {entries.length} 个字段</p>}
    </div>
  }
  return <span>—</span>
}
