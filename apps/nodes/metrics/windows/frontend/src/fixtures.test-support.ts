import type {MetricState} from './contracts'
import type {JSONValue} from './generated/variable-contract.generated'

export function metricState(metric: string, value?: JSONValue, status = 'ready', message?: string): MetricState {
  const type = typeof value === 'boolean' ? 'boolean' : typeof value === 'string' ? 'string' : 'number'
  return {metric, state: {
    version: 1, instance_id: '0123456789abcdef0123456789abcdef', revision: '9007199254740993',
    definition: {type, ...(type === 'number' ? {unit: '%', minimum: 0, maximum: 100} : {})},
    ...(value === undefined || status === 'unavailable' ? {} : {value, source_time_unix_ms: 1000}),
    status, observed_at_unix_ms: 1001,
    ...(message ? {error: {code: 'collection_failed', message}} : {}),
  }}
}
