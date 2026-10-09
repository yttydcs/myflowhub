import * as React from 'react'
import { cn } from '../../lib/utils'
import { guardDisabledButton, useFieldsetDisabled } from './control-events'
import { Select as ShadcnSelect, SelectContent, SelectItem, SelectTrigger, SelectValue } from './shadcn/select'

export interface SelectOption {
  value: string
  label: React.ReactNode
  disabled?: boolean
}

export interface SelectProps extends Pick<React.ButtonHTMLAttributes<HTMLButtonElement>, 'disabled' | 'id' | 'className' | 'aria-label' | 'aria-labelledby' | 'aria-invalid' | 'aria-describedby' | 'aria-busy'> {
  value: string
  onValueChange: (value: string) => void
  options: SelectOption[]
  placeholder?: string
}

export function Select({ value, onValueChange, options, disabled, className, placeholder, ...props }: SelectProps) {
  const trigger = React.useRef<HTMLButtonElement>(null)
  const fieldsetDisabled = useFieldsetDisabled(trigger)
  const [open, setOpen] = React.useState(false)
  React.useEffect(() => {
    if (disabled || fieldsetDisabled) setOpen(false)
  }, [disabled, fieldsetDisabled])
  const selectedIndex = options.findIndex((option) => option.value === value)
  const selected = options[selectedIndex]
  // All options are indexed, including "". Business values never enter Radix's
  // reserved-value namespace; option reorder recomputes the selected index.
  const internalValue = selectedIndex < 0 ? '' : String(selectedIndex)

  return <ShadcnSelect value={internalValue} disabled={disabled || fieldsetDisabled} open={open && !disabled && !fieldsetDisabled}
    onOpenChange={(next) => setOpen(next && !trigger.current?.matches(':disabled'))}
    onValueChange={(next) => {
      const option = options[Number(next)]
      if (next !== '' && option && !option.disabled && !trigger.current?.matches(':disabled')) onValueChange(option.value)
    }}>
    <SelectTrigger ref={trigger} size="sm" className={cn('select w-full min-w-0 rounded-md px-2.5 py-0 text-xs', className)}
      onPointerDown={guardDisabledButton} onKeyDown={guardDisabledButton} onClick={guardDisabledButton} {...props}>
      <SelectValue placeholder={selected ? undefined : (value || placeholder)}>{selected?.label}</SelectValue>
    </SelectTrigger>
    <SelectContent position="popper" align="start" sideOffset={4} collisionPadding={8}
      className="max-h-[min(280px,var(--radix-select-content-available-height))] max-w-[calc(100vw-16px)] rounded-lg data-[side=bottom]:translate-y-0 data-[side=top]:translate-y-0">
      {options.map((option, index) => <SelectItem key={option.value} value={String(index)} disabled={option.disabled} className="min-h-7 rounded-sm text-xs">
        {option.label}
      </SelectItem>)}
    </SelectContent>
  </ShadcnSelect>
}
