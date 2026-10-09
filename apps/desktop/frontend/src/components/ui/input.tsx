import * as React from 'react'
import { cn } from '../../lib/utils'
import { Input as ShadcnInput } from './shadcn/input'
import { Textarea as ShadcnTextarea } from './shadcn/textarea'

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(({ className, ...props }, ref) => (
  <ShadcnInput ref={ref} className={cn('input h-8 rounded-md px-2.5 text-xs md:text-xs', className)} {...props} />
))
Input.displayName = 'Input'

export const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(({ className, ...props }, ref) => (
  <ShadcnTextarea ref={ref} className={cn('input textarea rounded-md px-2.5 text-xs md:text-xs', className)} {...props} />
))
Textarea.displayName = 'Textarea'
