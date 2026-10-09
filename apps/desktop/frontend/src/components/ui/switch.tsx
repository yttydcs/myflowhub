import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'
import { guardDisabledButton } from './control-events'
import { Switch as ShadcnSwitch } from './shadcn/switch'

export interface SwitchProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange' | 'defaultChecked' | 'children'> {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}

export function Switch({ checked, onCheckedChange, className, onClick, onKeyDown, onPointerDown, ...props }: SwitchProps) {
  return <ShadcnSwitch checked={checked} onCheckedChange={onCheckedChange} className={cn('switch', className)}
    onClick={(event) => guardDisabledButton(event, onClick)}
    onKeyDown={(event) => guardDisabledButton(event, onKeyDown)}
    onPointerDown={(event) => guardDisabledButton(event, onPointerDown)} {...props} />
}
