# Flow

## Purpose

Flow 保存有界的自动化定义，调用已有资源，并通过订阅暴露运行状态和事件。

## Observable behavior

- definition 与 status 是 Variables；create/update/run/cancel/archive 是 Commands；run events 是 Stream。
- 定义有 schema version、稳定 flow ID、节点/边数量和输入大小上限。
- run 具有唯一 ID、dedupe key、deadline、并发上限、取消状态和有界输出。
- retry 只对声明为可重试的错误生效，使用受限 backoff；归档不会改变仍在运行的实例。

## Permissions

查看、修改、执行、取消、归档分别授权。Flow 代表发起主体调用资源，不能把服务器本地身份当成权限提升路径。

## Non-goals

- 不恢复通用 action dispatcher 或任意代码执行。
- 不在 Flow 内复制 Resource、Subscription 或 Command 的路由与权限逻辑。

## Acceptance

定义验证、dedupe、并发、取消、retry、权限、输出溢出、状态恢复和归档测试通过；删除或更新期间不存在孤儿运行实例。
