export type Page = 'overview' | 'nodes' | 'resources' | 'metrics' | 'clipboard' | 'files' | 'flows' | 'authority' | 'settings' | 'logs'

export const pages: Array<{ id: Page; label: string; hint: string }> = [
  { id: 'overview', label: '运行概览', hint: '连接与健康' },
  { id: 'nodes', label: '节点树', hint: '拓扑与资源' },
  { id: 'resources', label: '资源浏览器', hint: '变量 · 流 · 指令' },
  { id: 'metrics', label: '设备指标', hint: 'MetricsNode' },
  { id: 'clipboard', label: '剪贴板', hint: 'ClipboardNode' },
  { id: 'files', label: '文件传输', hint: 'File feature' },
  { id: 'flows', label: '自动化流', hint: 'Flow feature' },
  { id: 'authority', label: '权限与准入', hint: '服务端裁决' },
  { id: 'settings', label: '连接设置', hint: '版本化配置' },
  { id: 'logs', label: '操作日志', hint: '不含业务正文' },
]

export function pretty(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
