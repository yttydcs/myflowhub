import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export function safeJSON(value: string | undefined): unknown {
  if (!value) return null
  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

export function resourceKey(ownerNodeID: string, name: string): string {
  return `${ownerNodeID}:${name}`
}
