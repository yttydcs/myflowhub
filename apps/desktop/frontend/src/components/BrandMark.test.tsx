import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { BrandMark } from './BrandMark'

describe('BrandMark', () => {
  it('selects the full and compact V5 assets by display size', () => {
    const { container, rerender } = render(<BrandMark />)
    expect(container.querySelector('img')).toHaveAttribute('src', '/brand/myflowhub-symbol-v5.svg')

    rerender(<BrandMark size="compact" />)
    expect(container.querySelector('img')).toHaveAttribute('src', '/brand/myflowhub-symbol-v5-compact.svg')
    expect(container.querySelector('img')).toHaveClass('small')
  })
})
