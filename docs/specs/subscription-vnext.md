# Subscription vNext

> Partially superseded: Variable/Stream 的历史 delivery 语义仍是迁移基线；通用 observable、
> Topic 和 descriptor-driven authorization 以 [Resource Platform v2](resource-platform-v2.md) 为准。

Subscription is a first-class runtime relation over a Variable or Stream. Each relation records its ID, subscriber, resource, link binding, lease deadline, authorization deadline, topology epoch, policy generation, next hop, and bounded queue size.

## Variable delivery

Registration and snapshot capture are atomic at the resource boundary. The dispatcher always queues the snapshot before changes. If a consumer is slow, unsent Variable changes coalesce to the newest revision; the initial snapshot is never replaced.

## Stream delivery

Stream events retain sequence order while capacity is available. A source sequence jump produces a gap before the later event. Once pending capacity is exhausted, later events are summarized into an explicit gap range. The dispatcher sends the gap before accepting later events again. Silent loss is forbidden.

## Lifecycle

- lease and authorization deadlines must both cover the requested lifetime;
- unsubscribe closes the local relation and removes its interest;
- link cleanup removes all link-bound relations;
- lease expiry emits an observable expiry event and then closes delivery;
- the authority that promotes a cross-subtree request retains a bounded, payload-free forwarding record; a policy-generation change sends an observable expiry to the subscriber and a parent-control unsubscribe to the resource owner;
- manager shutdown cancels blocked dispatchers without leaking goroutines.

Intermediate interest aggregation suppresses duplicate upstream ownership work, but every subscriber's identity, link, authorization deadline, policy generation, and topology epoch remain distinct. Aggregation never grants one subscriber another subscriber's authority.

## Performance acceptance

The representative cross-subtree gate sends 50 sequential Variable updates across four Node links and measures publish-to-receive latency after the initial snapshot. Memory transport must remain below 100 ms p95 and 1 s total; TCP loopback must remain below 250 ms p95 and 2 s total. `tests/integration/TestCrossSubtreeSubscriptionLatencyBudget` owns this reproducible threshold. These are regression budgets, not production network SLOs.

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
