import { useRef, type AriaAttributes } from 'react'
import { cn } from '../../lib/utils'
import { Slider as ShadcnSlider } from './shadcn/slider'
import { isWithinDisabledFieldset, useFieldsetDisabled } from './control-events'

export interface SliderProps extends Pick<AriaAttributes, 'aria-label' | 'aria-labelledby' | 'aria-describedby' | 'aria-valuetext' | 'aria-busy'> {
  value: number
  onValueChange: (value: number) => void
  onValueCommit?: (value: number) => void
  min?: number
  max?: number
  step?: number
  disabled?: boolean
  id?: string
  className?: string
}

export function Slider({ value, onValueChange, onValueCommit, min = 0, max = 100, step = 1, disabled, id, className, ...aria }: SliderProps) {
  const root = useRef<HTMLSpanElement>(null)
  const fieldsetDisabled = useFieldsetDisabled(root)
  return <ShadcnSlider ref={root} value={[value]} onValueChange={([next]) => {
    if (next !== undefined && !disabled && !isWithinDisabledFieldset(root.current)) onValueChange(next)
  }} onValueCommit={([next]) => {
    // Radix keyboard commits precede onValueChange; consumers must receive
    // the new draft before the commit, just as they do for pointer release.
    queueMicrotask(() => {
      if (root.current && next !== undefined && !disabled && !isWithinDisabledFieldset(root.current)) onValueCommit?.(next)
    })
  }} onKeyDown={(event) => { if (isWithinDisabledFieldset(root.current)) event.preventDefault() }}
    onPointerDown={(event) => { if (isWithinDisabledFieldset(root.current)) event.preventDefault() }}
    min={min} max={max} step={step} disabled={disabled || fieldsetDisabled} className={cn('slider', className)} thumbProps={{ id, ...aria, 'aria-disabled': disabled || fieldsetDisabled || undefined }} />
}
