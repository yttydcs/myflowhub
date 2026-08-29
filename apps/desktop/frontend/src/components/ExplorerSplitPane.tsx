import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from 'react'
import {
  clampExplorerSplitRatio,
  DEFAULT_EXPLORER_SPLIT_RATIO,
  EXPLORER_SPLITTER_SIZE,
  MAX_EXPLORER_SPLIT_RATIO,
  MIN_EXPLORER_SPLIT_RATIO,
} from '../lib/explorer-split'

type Props = {
  ratio?: number
  onRatioChange(ratio: number): void
  topID: string
  bottomID: string
  top: ReactNode
  bottom: ReactNode
}

export function ExplorerSplitPane({
  ratio = DEFAULT_EXPLORER_SPLIT_RATIO,
  onRatioChange,
  topID,
  bottomID,
  top,
  bottom,
}: Props) {
  const rootRef = useRef<HTMLDivElement>(null)
  const pointerID = useRef<number>()
  const liveRatioRef = useRef(clampExplorerSplitRatio(ratio))
  const [liveRatio, setLiveRatio] = useState(liveRatioRef.current)

  useEffect(() => {
    if (pointerID.current !== undefined) return
    updateLiveRatio(clampExplorerSplitRatio(ratio, measuredHeight()))
  }, [ratio])

  function measuredHeight(): number {
    return rootRef.current?.getBoundingClientRect().height || 0
  }

  function updateLiveRatio(next: number) {
    liveRatioRef.current = next
    setLiveRatio(next)
  }

  function ratioFromClientY(clientY: number): number | undefined {
    const bounds = rootRef.current?.getBoundingClientRect()
    if (!bounds || bounds.height <= EXPLORER_SPLITTER_SIZE) return undefined
    const raw = (clientY - bounds.top) / (bounds.height - EXPLORER_SPLITTER_SIZE)
    return clampExplorerSplitRatio(raw, bounds.height)
  }

  function updateFromPointer(event: PointerEvent<HTMLDivElement>): number | undefined {
    const next = ratioFromClientY(event.clientY)
    if (next !== undefined) updateLiveRatio(next)
    return next
  }

  function handlePointerDown(event: PointerEvent<HTMLDivElement>) {
    if (event.button !== 0) return
    event.preventDefault()
    pointerID.current = event.pointerId
    event.currentTarget.setPointerCapture(event.pointerId)
    updateFromPointer(event)
  }

  function handlePointerMove(event: PointerEvent<HTMLDivElement>) {
    if (pointerID.current !== event.pointerId) return
    updateFromPointer(event)
  }

  function handlePointerUp(event: PointerEvent<HTMLDivElement>) {
    if (pointerID.current !== event.pointerId) return
    const next = updateFromPointer(event) ?? liveRatioRef.current
    pointerID.current = undefined
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    onRatioChange(next)
  }

  function handlePointerCancel(event: PointerEvent<HTMLDivElement>) {
    if (pointerID.current !== event.pointerId) return
    pointerID.current = undefined
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    updateLiveRatio(clampExplorerSplitRatio(ratio, measuredHeight()))
  }

  function commitRatio(next: number) {
    const clamped = clampExplorerSplitRatio(next, measuredHeight())
    updateLiveRatio(clamped)
    onRatioChange(clamped)
  }

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const step = event.shiftKey ? 0.1 : 0.03
    if (event.key === 'ArrowUp') commitRatio(liveRatioRef.current - step)
    else if (event.key === 'ArrowDown') commitRatio(liveRatioRef.current + step)
    else if (event.key === 'Home') commitRatio(MIN_EXPLORER_SPLIT_RATIO)
    else if (event.key === 'End') commitRatio(MAX_EXPLORER_SPLIT_RATIO)
    else return
    event.preventDefault()
  }

  function resetRatio() {
    commitRatio(DEFAULT_EXPLORER_SPLIT_RATIO)
  }

  return (
    <div
      ref={rootRef}
      className="explorer-split-pane"
      style={{ gridTemplateRows: `minmax(0, ${liveRatio}fr) ${EXPLORER_SPLITTER_SIZE}px minmax(0, ${1 - liveRatio}fr)` }}
    >
      <div id={topID} className="explorer-split-section">{top}</div>
      <div
        className="explorer-splitter"
        role="separator"
        aria-label="调整节点列表和资源列表高度"
        aria-orientation="horizontal"
        aria-controls={`${topID} ${bottomID}`}
        aria-valuemin={Math.round(MIN_EXPLORER_SPLIT_RATIO * 100)}
        aria-valuemax={Math.round(MAX_EXPLORER_SPLIT_RATIO * 100)}
        aria-valuenow={Math.round(liveRatio * 100)}
        aria-valuetext={`节点列表 ${Math.round(liveRatio * 100)}%，资源列表 ${Math.round((1 - liveRatio) * 100)}%`}
        tabIndex={0}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerCancel}
        onKeyDown={handleKeyDown}
        onDoubleClick={resetRatio}
      >
        <span aria-hidden="true" />
      </div>
      <div id={bottomID} className="explorer-split-section">{bottom}</div>
    </div>
  )
}
