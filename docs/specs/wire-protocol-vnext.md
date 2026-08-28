# Wire Protocol vNext

> Superseded: MFH3 fixed-operation envelope 已由 [Resource Platform v2](resource-platform-v2.md)
> 和 [Resource Sessions v2](resource-sessions-v2.md) 定义的 MFH4 v2 通用资源操作取代。本文仅保留迁移前参考。

## Status

Implemented phase-one contract. Incompatible with the former SubProto envelope.

## Frame

Every frame starts with an 88-byte big-endian header containing magic `MFH3`, protocol version, phase, operation, metadata lengths, payload length, source/target IDs, topology epoch, deadline, message/correlation IDs, and resource owner. The header is followed by content type, schema, resource local name, and the opaque payload.

Decoders validate all declared lengths before allocation. The default payload limit is 1 MiB and callers may configure a smaller limit. Short reads, short writes, invalid magic, unsupported versions, illegal phase/operation combinations, invalid resource names, and oversize payloads fail explicitly.

## Routing contract

- `request` travels toward a point that can adjudicate the caller and target.
- after authorization, the request becomes `control` and only travels down the authoritative tree.
- `response` carries a non-zero correlation ID.
- `event` carries resource changes, route lifecycle, liveness, or observable stream gaps.

The core operations are join/ack, route announce/withdraw, subscribe/unsubscribe/ack, variable snapshot/update, stream event/gap, command call/result, structured error, and heartbeat. There is no SubProto selector or handler dispatch field. Join claims sign their proposed topology epoch so reparent activation cannot substitute a stale generation. The child commits that exact signed epoch after verifying the acknowledgement; cleanup from an older session must match both the peer identity and its older epoch, so it cannot withdraw a replacement edge.

## Payloads and errors

The envelope does not interpret business payload bytes. `ContentType` and `Schema` make their interpretation explicit and independently versionable. Wire errors use stable codes such as `malformed`, `unauthenticated`, `forbidden`, `not_found`, `conflict`, `stale_epoch`, `expired`, `overflow`, `timeout`, and `internal`.

## Limits

- resource local name: 255 UTF-8 bytes, relative non-empty segments
- content type: 127 bytes
- schema: 255 bytes
- default payload: 1 MiB

Transport fragmentation is below this contract. A transport may use a smaller configured payload limit but may not silently truncate a valid frame.

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
