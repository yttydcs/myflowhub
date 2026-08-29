import { describe, expect, it } from 'vitest'
import {
  clampExplorerSplitRatio,
  DEFAULT_EXPLORER_SPLIT_RATIO,
  isExplorerSplitRatio,
} from './explorer-split'

describe('Explorer split bounds', () => {
  it('accepts only finite preference ratios inside the durable bounds', () => {
    expect(isExplorerSplitRatio(0.35)).toBe(true)
    expect(isExplorerSplitRatio(0.1)).toBe(false)
    expect(isExplorerSplitRatio(Number.NaN)).toBe(false)
    expect(isExplorerSplitRatio('0.35')).toBe(false)
  })

  it('clamps against both pane minimums and falls back safely without a measurement', () => {
    expect(clampExplorerSplitRatio(Number.NaN)).toBe(DEFAULT_EXPLORER_SPLIT_RATIO)
    expect(clampExplorerSplitRatio(0, 507)).toBeCloseTo(112 / 500)
    expect(clampExplorerSplitRatio(1, 507)).toBeCloseTo(1 - 144 / 500)
    expect(clampExplorerSplitRatio(0.2, 200)).toBeCloseTo(112 / (112 + 144))
  })
})
