import { describe, expect, it } from 'vitest'
import type { ViewDefinition, ViewLayoutNode, ViewWidget } from './types'
import {
  WorkspaceLayoutError,
  addWorkspaceWidget,
  dockWorkspaceView,
  firstWorkspaceLeafID,
  lastWorkspaceLeafID,
  removeWorkspaceWidget,
  resizeWorkspaceSplitPair,
  validateWorkspaceView,
  workspaceDropTargetPriority,
  workspaceLayoutMinimumSize,
  workspacePanelDockSide,
} from './workspace-layout'

const widget = (id: string): ViewWidget => ({
  id,
  owner_node_id: '2',
  resource_name: `system/${id}`,
  renderer: 'mfh.variable',
})

const view = (ids: string[], layout_root?: ViewLayoutNode): ViewDefinition => ({
  id: 'view',
  name: 'View',
  revision: 1,
  widgets: ids.map(widget),
  layout_root,
})

const leaf = (id: string): ViewLayoutNode => ({ kind: 'leaf', widget_id: id })

describe('workspace layout domain', () => {
  it('fills the first pane and appends direct additions at the workspace right', () => {
    const first = addWorkspaceWidget(view([]), widget('a'))
    expect(first.layout_root).toEqual(leaf('a'))
    const second = addWorkspaceWidget(first, widget('b'))
    expect(second.layout_root).toEqual({
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b')],
      weights: [0.5, 0.5],
    })
  })

  it('inserts a third sibling between two horizontal panes', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b'), leaf('c')],
      weights: [1 / 3, 1 / 3, 1 / 3],
    })
    const next = dockWorkspaceView(base, { kind: 'existing', widgetID: 'c' }, {
      kind: 'panel-edge',
      targetWidgetID: 'b',
      side: 'left',
    })
    expect((next.layout_root as Extract<ViewLayoutNode, { kind: 'split' }>).children).toEqual([leaf('a'), leaf('c'), leaf('b')])
  })

  it('wraps only the target for a cross-axis panel drop', () => {
    const base = view(['a', 'b'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b')],
      weights: [0.5, 0.5],
    })
    const next = dockWorkspaceView(base, { kind: 'new', widget: widget('c') }, {
      kind: 'panel-edge',
      targetWidgetID: 'b',
      side: 'bottom',
    })
    expect(next.layout_root).toEqual({
      kind: 'split',
      axis: 'horizontal',
      children: [
        leaf('a'),
        { kind: 'split', axis: 'vertical', children: [leaf('b'), leaf('c')], weights: [0.5, 0.5] },
      ],
      weights: [0.5, 0.5],
    })
  })

  it('splits the whole root at a workspace outer edge', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b'), leaf('c')],
      weights: [1 / 3, 1 / 3, 1 / 3],
    })
    const next = dockWorkspaceView(base, { kind: 'new', widget: widget('d') }, { kind: 'root-edge', side: 'bottom' })
    expect(next.layout_root?.kind).toBe('split')
    if (next.layout_root?.kind !== 'split') throw new Error('expected split root')
    expect(next.layout_root.axis).toBe('vertical')
    expect(next.layout_root.children[0]).toMatchObject({ kind: 'split', axis: 'horizontal' })
    expect(next.layout_root.children[1]).toEqual(leaf('d'))
    expect(next.layout_root.weights).toEqual([0.5, 0.5])
  })

  it('inserts at an exact divider and preserves customized unrelated weights', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b'), leaf('c')],
      weights: [0.2, 0.3, 0.5],
    })
    const next = dockWorkspaceView(base, { kind: 'new', widget: widget('d') }, {
      kind: 'split-gap',
      parentPath: [],
      insertionIndex: 1,
      beforeWidgetID: 'a',
      afterWidgetID: 'b',
    })
    expect(next.layout_root).toEqual({
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('d'), leaf('b'), leaf('c')],
      weights: [0.2, 0.15, 0.15, 0.5],
    })
  })

  it('rejects stale divider anchors and treats self targets as no-ops', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b'), leaf('c')],
      weights: [0.3, 0.3, 0.4],
    })
    expect(dockWorkspaceView(base, { kind: 'existing', widgetID: 'b' }, {
      kind: 'panel-edge', targetWidgetID: 'b', side: 'bottom',
    })).toBe(base)
    expect(() => dockWorkspaceView(base, { kind: 'new', widget: widget('d') }, {
      kind: 'split-gap',
      parentPath: [],
      insertionIndex: 1,
      beforeWidgetID: 'missing',
      afterWidgetID: 'b',
    })).toThrow(/no longer exists/)
  })

  it('moves a direct sibling with its existing weight', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b'), leaf('c')],
      weights: [0.2, 0.3, 0.5],
    })
    const next = dockWorkspaceView(base, { kind: 'existing', widgetID: 'c' }, {
      kind: 'panel-edge',
      targetWidgetID: 'b',
      side: 'left',
    })
    expect(next.layout_root).toEqual({
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('c'), leaf('b')],
      weights: [0.2, 0.5, 0.3],
    })
  })

  it('collapses removed branches and flattens the resulting same-axis split', () => {
    const base = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [
        leaf('a'),
        {
          kind: 'split',
          axis: 'vertical',
          children: [
            leaf('b'),
            { kind: 'split', axis: 'horizontal', children: [leaf('c'), leaf('a')], weights: [0.5, 0.5] },
          ],
          weights: [0.5, 0.5],
        },
      ],
      weights: [0.5, 0.5],
    })
    expect(() => validateWorkspaceView(base)).toThrow(WorkspaceLayoutError)

    const valid = view(['a', 'b', 'c'], {
      kind: 'split',
      axis: 'horizontal',
      children: [
        leaf('a'),
        { kind: 'split', axis: 'vertical', children: [leaf('b'), leaf('c')], weights: [0.5, 0.5] },
      ],
      weights: [0.5, 0.5],
    })
    const next = removeWorkspaceWidget(valid, 'b')
    expect(next.layout_root).toEqual({
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('c')],
      weights: [0.5, 0.5],
    })
  })

  it('resizes only an adjacent pair and computes recursive minimum size', () => {
    const root: ViewLayoutNode = {
      kind: 'split',
      axis: 'horizontal',
      children: [
        leaf('a'),
        { kind: 'split', axis: 'vertical', children: [leaf('b'), leaf('c')], weights: [0.5, 0.5] },
        leaf('d'),
      ],
      weights: [0.2, 0.5, 0.3],
    }
    expect(resizeWorkspaceSplitPair(root, [], 1, 0.25)).toMatchObject({ weights: [0.2, 0.2, 0.6] })
    expect(workspaceLayoutMinimumSize(root)).toEqual({ width: 616, height: 248 })
  })

  it('validates leaf equality, normalized weights and bounded input', () => {
    expect(() => validateWorkspaceView(view(['a'], leaf('missing')))).toThrow(/missing widget|match widgets/)
    expect(() => validateWorkspaceView(view(['a', 'b'], {
      kind: 'split',
      axis: 'horizontal',
      children: [leaf('a'), leaf('b')],
      weights: [0.5, Number.NaN],
    }))).toThrow(/finite/)
    expect(() => addWorkspaceWidget(view(['a'], leaf('a')), widget('a'))).toThrow(/already exists/)
    expect(() => dockWorkspaceView(view([]), { kind: 'new', widget: widget('a') }, { kind: 'root-edge', side: 'right' })).toThrow(/empty view/)
    expect(dockWorkspaceView(view(['a'], leaf('a')), { kind: 'existing', widgetID: 'a' }, { kind: 'root-edge', side: 'bottom' }))
      .toEqual(view(['a'], leaf('a')))
  })

  it('finds stable boundary leaves', () => {
    const root: ViewLayoutNode = {
      kind: 'split',
      axis: 'vertical',
      children: [leaf('a'), { kind: 'split', axis: 'horizontal', children: [leaf('b'), leaf('c')], weights: [0.5, 0.5] }],
      weights: [0.5, 0.5],
    }
    expect(firstWorkspaceLeafID(root)).toBe('a')
    expect(lastWorkspaceLeafID(root)).toBe('c')
  })

  it('keeps panel centers inactive and resolves edge/corner intent deterministically', () => {
    expect(workspacePanelDockSide(0.5, 0.5)).toBeUndefined()
    expect(workspacePanelDockSide(0.1, 0.5)).toBe('left')
    expect(workspacePanelDockSide(0.9, 0.5)).toBe('right')
    expect(workspacePanelDockSide(0.5, 0.1)).toBe('top')
    expect(workspacePanelDockSide(0.5, 0.9)).toBe('bottom')
    expect(workspacePanelDockSide(0.1, 0.1)).toBe('left')
    expect(() => workspacePanelDockSide(0.5, 0.5, 0.5)).toThrow(/invalid/)
  })

  it('prioritizes exact dividers, then outer edges, panels, and background', () => {
    expect([
      'workspace-drop',
      'workspace-panel:a',
      'workspace-root-edge:left',
      'workspace-divider:root:0',
    ].sort((left, right) => workspaceDropTargetPriority(left) - workspaceDropTargetPriority(right))).toEqual([
      'workspace-divider:root:0',
      'workspace-root-edge:left',
      'workspace-panel:a',
      'workspace-drop',
    ])
  })
})
