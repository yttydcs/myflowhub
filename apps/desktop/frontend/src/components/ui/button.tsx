import * as React from 'react'
import { cn } from '../../lib/utils'
import { Button as ShadcnButton } from './shadcn/button'

export type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'default' | 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'default' | 'md' | 'sm' | 'icon'
}

const variants = { default: 'default', primary: 'default', secondary: 'outline', ghost: 'ghost', danger: 'destructive' } as const
const sizes = { default: 'h-8 px-3 py-0', md: 'h-8 px-3 py-0', sm: 'h-7 px-2.5 py-0', icon: 'size-8 p-0' } as const

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant = 'default', size = 'default', ...props }, ref) => (
  <ShadcnButton ref={ref} variant={variants[variant]} size={size === 'md' ? 'default' : size} className={cn('button', `button-${variant === 'default' ? 'primary' : variant}`, `button-${size === 'default' ? 'md' : size}`, 'gap-1.5 rounded-md text-xs', sizes[size], className)} {...props} />
))
Button.displayName = 'Button'
