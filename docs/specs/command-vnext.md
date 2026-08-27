# Command vNext

Command is the imperative complement to Variable and Stream subscriptions. A call carries a unique message ID, caller identity, target resource, bounded input, and an absolute deadline.

## Authorization origin

The dispatcher accepts only two explicit origins:

- `adjudicated`: a request whose policy was evaluated at the LCA or owning authority;
- `parent-control`: a control frame already validated as coming from the current authenticated parent with the current topology epoch.

Merely arriving on a connection or declaring `control` is insufficient. Input, output, quotas, and handler validation still run for parent control.

## Lifecycle and safety

- active handlers are capped independently from pending callers;
- in-flight and recently completed message IDs are rejected as duplicates;
- completed IDs live in a bounded TTL/LRU cache;
- deadlines return a structured timeout even if a handler ignores cancellation;
- a late handler result is diagnostic only and cannot complete the request again;
- handler panic is recovered as a structured internal error;
- missing or non-command targets produce stable not-found/conflict results.

Command results and errors use the request message ID as their wire correlation ID. The node runtime owns cross-tree routing; the dispatcher never imports a concrete Transport.

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
