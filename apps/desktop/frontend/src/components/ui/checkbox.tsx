import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'
import { guardDisabledButton } from './control-events'
import { Checkbox as ShadcnCheckbox } from './shadcn/checkbox'

export interface CheckboxProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange' | 'defaultChecked' | 'children'> {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}

export function Checkbox({ checked, onCheckedChange, className, onClick, onKeyDown, onPointerDown, ...props }: CheckboxProps) {
  return <ShadcnCheckbox checked={checked} onCheckedChange={(next) => onCheckedChange(next === true)} className={cn('checkbox', className)}
    onClick={(event) => guardDisabledButton(event, onClick)}
    onKeyDown={(event) => guardDisabledButton(event, onKeyDown)}
    onPointerDown={(event) => guardDisabledButton(event, onPointerDown)} {...props} />
}
