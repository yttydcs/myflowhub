import * as React from 'react'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Select, type SelectOption } from './select'
import { Switch } from './switch'
import { Checkbox } from './checkbox'
import { Slider } from './slider'
import { Progress } from './progress'
import { Button } from './button'
import { Input, Textarea } from './input'

function SelectExample({ options, initial = '', onChange = () => {} }: { options: SelectOption[]; initial?: string; onChange?: (value: string) => void }) {
  const [value, setValue] = React.useState(initial)
  return <Select aria-label="Mode" value={value} options={options} onValueChange={(next) => { setValue(next); onChange(next) }} />
}

async function openSelect(key = 'ArrowDown') {
  const trigger = screen.getByRole('combobox')
  trigger.focus()
  fireEvent.keyDown(trigger, { key })
  return screen.findByRole('listbox')
}

describe('Desktop shadcn controls', () => {
  it('portals the menu, skips disabled options with arrows, commits with Enter and restores focus', async () => {
    const onChange = vi.fn()
    const { container } = render(<SelectExample initial="one" onChange={onChange} options={[
      { value: 'one', label: 'One' }, { value: 'disabled', label: 'Unavailable', disabled: true }, { value: 'two', label: 'Two' },
    ]} />)
    const list = await openSelect()
    expect(container.contains(list)).toBe(false)
    expect(within(list).getByRole('option', { name: 'Unavailable' })).toHaveAttribute('aria-disabled', 'true')
    await waitFor(() => expect(within(list).getByRole('option', { name: 'One' })).toHaveFocus())
    fireEvent.keyDown(document.activeElement!, { key: 'ArrowDown' })
    await waitFor(() => expect(within(list).getByRole('option', { name: 'Two' })).toHaveFocus())
    fireEvent.keyDown(document.activeElement!, { key: 'Enter' })
    await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
    expect(onChange).toHaveBeenCalledExactlyOnceWith('two')
    expect(screen.getByRole('combobox')).toHaveTextContent('Two')
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveFocus())
  })

  it('Escape cancels keyboard navigation without changing the selected value', async () => {
    const onChange = vi.fn()
    render(<SelectExample initial="one" onChange={onChange} options={[{ value: 'one', label: 'One' }, { value: 'two', label: 'Two' }]} />)
    const list = await openSelect('Enter')
    await waitFor(() => expect(within(list).getByRole('option', { name: 'One' })).toHaveFocus())
    fireEvent.keyDown(document.activeElement!, { key: 'ArrowDown' })
    await waitFor(() => expect(within(list).getByRole('option', { name: 'Two' })).toHaveFocus())
    fireEvent.keyDown(document.activeElement!, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole('combobox')).toHaveTextContent('One')
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveFocus())
  })

  it('round-trips empty and sentinel-looking business values through indexed options', async () => {
    const onChange = vi.fn()
    render(<SelectExample onChange={onChange} options={[
      { value: '', label: 'Empty string' }, { value: '0', label: 'Zero string' }, { value: '__empty__', label: 'Sentinel string' },
    ]} />)
    expect(screen.getByRole('combobox')).toHaveTextContent('Empty string')
    let list = await openSelect()
    fireEvent.click(within(list).getByRole('option', { name: 'Zero string' }))
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveTextContent('Zero string'))
    expect(onChange).toHaveBeenLastCalledWith('0')
    list = await openSelect()
    fireEvent.click(within(list).getByRole('option', { name: 'Sentinel string' }))
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveTextContent('Sentinel string'))
    expect(onChange).toHaveBeenLastCalledWith('__empty__')
    list = await openSelect()
    fireEvent.click(within(list).getByRole('option', { name: 'Empty string' }))
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveTextContent('Empty string'))
    expect(onChange.mock.calls.map(([value]) => value)).toEqual(['0', '__empty__', ''])
  })

  it('keeps absent values visible without selecting a fallback, including after reorder', async () => {
    const onChange = vi.fn()
    const options = [{ value: 'a', label: <strong>Alpha</strong> }, { value: 'b', label: 'Beta' }]
    const { rerender } = render(<Select value="missing" options={options} onValueChange={onChange} aria-label="Mode" placeholder="Choose" />)
    expect(screen.getByRole('combobox')).toHaveTextContent('missing')
    const list = await openSelect()
    expect(within(list).getAllByRole('option').every((option) => option.getAttribute('aria-selected') === 'false')).toBe(true)
    fireEvent.keyDown(list, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
    rerender(<Select value="b" options={[...options].reverse()} onValueChange={onChange} aria-label="Mode" />)
    expect(screen.getByRole('combobox')).toHaveTextContent('Beta')
    rerender(<Select value="" options={[]} onValueChange={onChange} aria-label="Mode" placeholder="Choose" />)
    expect(screen.getByRole('combobox')).toHaveTextContent('Choose')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('exposes label association, descriptions, invalid state and disabled state on the trigger', () => {
    render(<><label htmlFor="mode">Mode label</label><p id="hint">Choose a mode</p><Select id="mode" value="a" options={[{ value: 'a', label: 'Alpha' }]} onValueChange={vi.fn()} disabled aria-invalid aria-describedby="hint" /></>)
    const trigger = screen.getByRole('combobox', { name: 'Mode label' })
    expect(trigger).toBeDisabled()
    expect(trigger).toHaveAccessibleDescription('Choose a mode')
    expect(trigger).toHaveAttribute('aria-invalid', 'true')
    fireEvent.pointerDown(trigger, { button: 0 })
    fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('honors disabled ancestor fieldsets for Select, Switch and Checkbox', () => {
    const onChange = vi.fn()
    render(<fieldset disabled><Select value="a" options={[{ value: 'a', label: 'Alpha' }]} onValueChange={onChange} aria-label="Mode" /><Switch checked={false} onCheckedChange={onChange} aria-label="Power" /><Checkbox checked={false} onCheckedChange={onChange} aria-label="Follow" /></fieldset>)
    for (const role of ['combobox', 'switch', 'checkbox']) {
      const control = screen.getByRole(role)
      expect(control).toBeDisabled()
      fireEvent.pointerDown(control, { button: 0 })
      fireEvent.keyDown(control, { key: role === 'combobox' ? 'ArrowDown' : ' ' })
      fireEvent.click(control)
    }
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it.each([['switch', Switch], ['checkbox', Checkbox]] as const)('%s emits only booleans and does not submit its form', (role, Control) => {
    const onChange = vi.fn()
    const onSubmit = vi.fn((event: React.FormEvent) => event.preventDefault())
    const { rerender } = render(<form onSubmit={onSubmit}><Control checked={false} onCheckedChange={onChange} aria-label="Toggle" /></form>)
    fireEvent.click(screen.getByRole(role))
    expect(onChange).toHaveBeenCalledExactlyOnceWith(true)
    expect(onSubmit).not.toHaveBeenCalled()
    rerender(<Control checked onCheckedChange={onChange} aria-label="Toggle" />)
    expect(screen.getByRole(role)).toBeChecked()
    fireEvent.click(screen.getByRole(role))
    expect(onChange).toHaveBeenLastCalledWith(false)
    rerender(<Control checked disabled onCheckedChange={onChange} aria-label="Toggle" />)
    fireEvent.click(screen.getByRole(role))
    expect(onChange).toHaveBeenCalledTimes(2)
  })

  it('closes an open Select when its fieldset is disabled without reopening on enable', async () => {
    const onChange = vi.fn()
    const control = <Select value="a" options={[{ value: 'a', label: 'Alpha' }, { value: 'b', label: 'Beta' }]} onValueChange={onChange} aria-label="Mode" />
    const { rerender } = render(<fieldset>{control}</fieldset>)
    await openSelect()
    rerender(<fieldset disabled>{control}</fieldset>)
    await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument())
    rerender(<fieldset>{control}</fieldset>)
    await waitFor(() => expect(screen.getByRole('combobox')).not.toBeDisabled())
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('labels the Slider thumb and handles scalar keyboard steps and boundaries', () => {
    const onChange = vi.fn()
    function Example() {
      const [value, setValue] = React.useState(4)
      return <><span id="volume">Volume</span><Slider aria-labelledby="volume" value={value} min={0} max={10} step={2} onValueChange={(next) => { setValue(next); onChange(next) }} /></>
    }
    render(<Example />)
    const thumb = screen.getByRole('slider', { name: 'Volume' })
    thumb.focus()
    fireEvent.keyDown(thumb, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenLastCalledWith(6)
    expect(thumb).toHaveAttribute('aria-valuenow', '6')
    fireEvent.keyDown(thumb, { key: 'End' })
    expect(onChange).toHaveBeenLastCalledWith(10)
    fireEvent.keyDown(thumb, { key: 'Home' })
    expect(onChange).toHaveBeenLastCalledWith(0)
    fireEvent.keyDown(thumb, { key: 'ArrowLeft' })
    expect(onChange).toHaveBeenCalledTimes(3)
  })

  it('does not change a disabled Slider through pointer or keyboard input', () => {
    const onChange = vi.fn()
    render(<Slider value={4} onValueChange={onChange} aria-label="Volume" disabled />)
    const thumb = screen.getByRole('slider', { name: 'Volume' })
    expect(thumb).not.toHaveAttribute('tabindex', '0')
    fireEvent.keyDown(thumb, { key: 'ArrowRight' })
    fireEvent.pointerDown(thumb, { button: 0 })
    expect(onChange).not.toHaveBeenCalled()
  })

  it('disables Slider interaction and focus when its ancestor fieldset becomes disabled', async () => {
    const onChange = vi.fn()
    const { rerender } = render(<fieldset><Slider value={4} onValueChange={onChange} aria-label="Volume" /></fieldset>)
    expect(screen.getByRole('slider')).toHaveAttribute('tabindex', '0')
    rerender(<fieldset disabled><Slider value={4} onValueChange={onChange} aria-label="Volume" /></fieldset>)
    const thumb = screen.getByRole('slider')
    fireEvent.keyDown(thumb, { key: 'ArrowRight' })
    fireEvent.pointerDown(thumb, { button: 0 })
    await waitFor(() => expect(thumb).toHaveAttribute('aria-disabled', 'true'))
    expect(thumb).not.toHaveAttribute('tabindex', '0')
    expect(onChange).not.toHaveBeenCalled()
    rerender(<fieldset><Slider value={4} onValueChange={onChange} aria-label="Volume" /></fieldset>)
    await waitFor(() => expect(thumb).toHaveAttribute('tabindex', '0'))
    fireEvent.keyDown(thumb, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledExactlyOnceWith(5)
  })

  it('preserves the first-legend exception inside a disabled fieldset', () => {
    const onChange = vi.fn()
    render(<fieldset disabled><legend><Slider value={4} onValueChange={onChange} aria-label="Volume" /></legend></fieldset>)
    fireEvent.keyDown(screen.getByRole('slider'), { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledExactlyOnceWith(5)
  })

  it('renders styled Progress with the same bounded ratio in its ARIA and indicator', () => {
    const { container, rerender } = render(<Progress value={25} max={50} aria-label="Upload" />)
    expect(container.querySelector('progress')).toBeNull()
    expect(screen.getByRole('progressbar', { name: 'Upload' })).toHaveAttribute('aria-valuenow', '25')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuemax', '50')
    expect(container.querySelector('[data-slot="progress-indicator"]')).toHaveStyle({ transform: 'translateX(-50%)' })
    rerender(<Progress value={80} max={50} aria-label="Upload" />)
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '50')
    expect(container.querySelector('[data-slot="progress-indicator"]')).toHaveStyle({ transform: 'translateX(-0%)' })
  })

  it('retains Input/Textarea refs and existing Button variants while using shadcn sources', () => {
    const input = React.createRef<HTMLInputElement>()
    const textarea = React.createRef<HTMLTextAreaElement>()
    const button = React.createRef<HTMLButtonElement>()
    render(<><Input ref={input} aria-label="Name" /><Textarea ref={textarea} aria-label="Details" /><Button ref={button} variant="danger" size="sm">Delete</Button><Button variant="primary" size="md">Save</Button></>)
    expect(input.current).toBe(screen.getByRole('textbox', { name: 'Name' }))
    expect(textarea.current).toBe(screen.getByRole('textbox', { name: 'Details' }))
    expect(button.current).toBe(screen.getByRole('button', { name: 'Delete' }))
    expect(button.current).toHaveAttribute('data-variant', 'destructive')
    expect(screen.getByRole('button', { name: 'Save' })).toHaveAttribute('data-slot', 'button')
  })
})
