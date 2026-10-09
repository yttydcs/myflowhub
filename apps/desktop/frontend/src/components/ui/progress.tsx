import type { HTMLAttributes } from 'react'
import { cn } from '../../lib/utils'
import { Progress as ShadcnProgress } from './shadcn/progress'

export interface ProgressProps extends HTMLAttributes<HTMLDivElement> {
  value: number
  max: number
  'aria-label': string
}

export function Progress({ value, max, className, ...props }: ProgressProps) {
  const boundedMax = Number.isFinite(max) && max > 0 ? max : 1
  const boundedValue = Number.isFinite(value) ? Math.min(boundedMax, Math.max(0, value)) : 0
  return <ShadcnProgress value={boundedValue} max={boundedMax} className={cn('progress h-1.5', className)} {...props} />
}
