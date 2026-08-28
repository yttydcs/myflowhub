# File transfer

## 定位

File 是 Node-owned `mfh.file` Resource，不是独立子协议，也不再以 offer/chunk/complete/cancel 四个
Command 暴露。客户端对 `file/upload` 打开绑定 subject、resource、capability、link、topology epoch、
policy generation 和 expiry 的 session；控制消息与有界 data lane 共同使用 canonical wire。

## 资源

- `file/transfers`：Variable，提供当前与历史 transfer 摘要；
- `file/progress`：Stream，提供进度、完成、取消和失败事件；
- `file/upload`：`mfh.file` session Resource，以 `open` capability 创建上传会话。

open payload 使用 `mfh.file.offer.v1`，返回 grant 与初始 `mfh.file.progress.v1`。每个 data 消息包含
session ID、offset、data 和 SHA-256；close 选择 commit 或 abort，并返回最终 progress。SDK 的
`Files(...).UploadFile` 和 Desktop File renderer 封装这一流程。

## 行为与边界

- chunk 大小、单文件大小、总活动字节、并发 session、历史记录和有效期全部有界；
- offset 必须连续；每块 checksum、最终 size 与完整文件 digest 必须一致；
- 内容先写入配置根下的临时文件，完整验证后原子 rename 到目标；
- 零字节文件仍经过 open/close commit；过期、撤权、reparent、link 关闭和显式 abort 会清理 session
  与临时状态并发布失败或取消结果；
- control/data 使用独立有界队列和公平优先调度，大文件不能无限占用控制队列。

## 权限与路径

authority 对 `file/upload` 的 exact `open` capability 执行 `file.write` 裁决；owner 在 session open、
data 和 close 各阶段继续验证绑定与业务约束。路径固定解析在 File 配置根内，拒绝绝对路径、dot
segment、非法 UTF-8、symlink escape 和非普通源文件。

## 非目标

- 不提供任意远程文件系统代理；
- 不把文件块作为普通 Stream/Topic event；
- 不提供旧 Command chunk compatibility path；
- Media/WebRTC/编解码与平台 capture 属于独立后续计划。

完整 session 契约见 [Resource Sessions v2](../specs/resource-sessions-v2.md)。
