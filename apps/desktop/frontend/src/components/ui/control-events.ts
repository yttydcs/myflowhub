import { useEffect, useState, type RefObject, type SyntheticEvent } from 'react'

// Radix handles pointer/key events before native fieldset click suppression.
// :disabled preserves the HTML exception for the fieldset's first legend.
export function guardDisabledButton<E extends SyntheticEvent<HTMLButtonElement>>(event: E, handler?: (event: E) => void) {
  if (event.currentTarget.matches(':disabled')) event.preventDefault()
  else handler?.(event)
}

export function isWithinDisabledFieldset(element: Element | null): boolean {
  for (let fieldset = element?.closest('fieldset[disabled]'); fieldset; fieldset = fieldset.parentElement?.closest('fieldset[disabled]')) {
    const legend = Array.from(fieldset.children).find((child) => child.tagName === 'LEGEND')
    if (!legend?.contains(element)) return true
  }
  return false
}

export function useFieldsetDisabled(ref: RefObject<HTMLElement>) {
  const [disabled, setDisabled] = useState(false)
  useEffect(() => {
    const update = () => setDisabled(isWithinDisabledFieldset(ref.current))
    update()
    const observer = new MutationObserver(update)
    for (let parent = ref.current?.parentElement; parent; parent = parent.parentElement) {
      if (parent.tagName === 'FIELDSET') observer.observe(parent, { attributes: true, attributeFilter: ['disabled'] })
    }
    return () => observer.disconnect()
  }, [ref])
  return disabled
}
