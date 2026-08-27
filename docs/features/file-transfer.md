# File transfer

## Purpose

File 以受控会话组合 Commands、Variable/Stream 和有界 chunk call，支持跨节点文件传输而不引入独立子协议。

## Observable behavior

- offer/start、chunk、complete、cancel 是带 transfer ID 的 Commands。
- 状态或结果可读为 Variable，进度与完成事件通过 Stream 订阅。
- chunk 带 offset、长度、checksum；重复同一 chunk 幂等，gap、冲突内容或越界明确失败。
- 写入临时文件，完整尺寸和 digest 验证后原子落盘；取消或失败清理临时状态。

## Permissions

offer、read、write、list 和管理目标目录分别授权。路径解析固定在配置根内，拒绝绝对路径、dot segment、symlink escape 与非法 UTF-8。

## Non-goals

- 不提供任意远程文件系统代理。
- 不让大文件占用无界 envelope、内存或控制队列。

## Acceptance

零字节、大文件、重复/gap chunk、checksum、取消、重启清理、路径逃逸与跨权限测试通过。
