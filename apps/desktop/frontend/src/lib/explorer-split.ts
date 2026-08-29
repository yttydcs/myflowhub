export const DEFAULT_EXPLORER_SPLIT_RATIO = 0.35
export const MIN_EXPLORER_SPLIT_RATIO = 0.2
export const MAX_EXPLORER_SPLIT_RATIO = 0.8
export const EXPLORER_SPLITTER_SIZE = 7
export const MIN_NODE_PANE_PX = 112
export const MIN_RESOURCE_PANE_PX = 144

export function isExplorerSplitRatio(value: unknown): value is number {
  return typeof value === 'number'
    && Number.isFinite(value)
    && value >= MIN_EXPLORER_SPLIT_RATIO
    && value <= MAX_EXPLORER_SPLIT_RATIO
}

export function clampExplorerSplitRatio(
  value: number,
  containerHeight = 0,
  minimumNodeHeight = MIN_NODE_PANE_PX,
  minimumResourceHeight = MIN_RESOURCE_PANE_PX,
): number {
  const finiteValue = Number.isFinite(value) ? value : DEFAULT_EXPLORER_SPLIT_RATIO
  const preferred = clamp(finiteValue, MIN_EXPLORER_SPLIT_RATIO, MAX_EXPLORER_SPLIT_RATIO)
  const available = containerHeight - EXPLORER_SPLITTER_SIZE
  if (!Number.isFinite(available) || available <= 0) return preferred

  const minimumTotal = minimumNodeHeight + minimumResourceHeight
  if (available <= minimumTotal) {
    return clamp(minimumNodeHeight / minimumTotal, MIN_EXPLORER_SPLIT_RATIO, MAX_EXPLORER_SPLIT_RATIO)
  }

  const minimum = Math.max(MIN_EXPLORER_SPLIT_RATIO, minimumNodeHeight / available)
  const maximum = Math.min(MAX_EXPLORER_SPLIT_RATIO, 1 - minimumResourceHeight / available)
  return clamp(preferred, minimum, maximum)
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}
