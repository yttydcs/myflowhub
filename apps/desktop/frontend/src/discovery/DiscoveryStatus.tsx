import { createContext, useContext } from 'react'
import type { CatalogState, LoadState } from './controller'

export function DiscoveryStatus({ state, label, onRetry }: { state?: LoadState; label: string; onRetry(): void }) {
  if (state?.status === 'loaded' && !state.stale) return null
  return <div className="discovery-status" role={state?.status === 'error' ? 'alert' : 'status'}>
    <span>{state?.status === 'error' ? `${label}：${state.error}` : state?.status === 'loading' ? `正在加载${label}…` : `${label}尚未加载`}{state?.stale && '（显示缓存，待刷新）'}</span>
    {state?.status !== 'loading' && <button type="button" onClick={onRetry}>重试{label}</button>}
  </div>
}

export const CatalogContext = createContext<{ states: ReadonlyMap<string, CatalogState>; retry(owner: string): void } | undefined>(undefined)

export function MissingResource({ owner, name }: { owner: string; name: string }) {
  const context = useContext(CatalogContext)
  const state = context?.states.get(owner)
  return <div className="missing-resource">
    {context && state?.status !== 'loaded'
      ? <DiscoveryStatus state={state} label={`Node ${owner} 资源目录`} onRetry={() => context.retry(owner)} />
      : <strong>资源暂不可用</strong>}
    <p>保留布局，等待 {owner}/{name} 恢复。</p>
    {context && state?.status === 'loaded' && <button type="button" onClick={() => context.retry(owner)}>刷新资源目录</button>}
  </div>
}
