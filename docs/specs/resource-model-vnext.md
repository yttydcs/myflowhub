# Resource Model vNext

## Ownership and addressing

A resource is owned by exactly one node and addressed by `(NodeID, local name)`. Resource names are relative UTF-8 paths with non-empty segments. They do not form another authority or routing tree.

The owner registers a descriptor containing the resource kind, content type, schema, permission metadata, and a value-size limit. Duplicate local names and foreign owners fail explicitly.

## Variable

A Variable always has a current byte value and a non-zero, monotonically increasing revision. A subscriber receives a snapshot before later changes. Applying an equal or lower revision is rejected; values are copied at the API boundary and bounded by the descriptor.

## Stream

A Stream is a sequenced event source and does not imply a current value. Each publish increases the sequence. Applying an equal or lower sequence is rejected. Sequence gaps are preserved for the subscription layer to expose explicitly.

## Command

A Command is the imperative complement to subscription. Its descriptor and handler validate bounded input and output. Deadline, deduplication, policy adjudication, panic isolation, and remote result correlation belong to the command runtime rather than the resource registry.

## Non-goals

File, Flow, Variable, Stream, and Command do not each receive a subprotocol or Go module. Higher-level features compose these primitives and remain subject to the same node tree and policy path.

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
