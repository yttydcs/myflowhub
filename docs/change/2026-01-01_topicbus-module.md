# TopicBus module (Wails)

## Background / Goal
Deliver the TopicBus module in the Wails UI with subscribe/publish controls, topic filtering, event stream display, and profile-aware preference storage while reusing server protocol and service logic.

## Changes
- Added profile-aware TopicBus preferences storage (topics list + max events) in `MyFlowHub-Win/app_topicbus.go`.
- Extended `MyFlowHub-Win/internal/services/topicbus/service.go` with `*Simple` methods to align with frontend bindings.
- Added TopicBus frontend store and UI:
  - `MyFlowHub-Win/frontend/src/stores/topicbus.ts`
  - `MyFlowHub-Win/frontend/src/pages/TopicBus.vue`
- Routed `/topicbus` to the new module page in `MyFlowHub-Win/frontend/src/router/index.ts`.

## Task Mapping
- T8 TopicBus module: implemented subscribe/publish, event stream, filters, max events, and profile-aware persistence.

## Design Decisions / Tradeoffs
- Stored TopicBus subscriptions and max events per profile using the existing settings store, preserving legacy key names (`topicbus.subs`, `topicbus.max_events`) for migration compatibility.
- Throttled UI event list updates (200ms) to limit render pressure while preserving event order.
- Added `*Simple` service wrappers to avoid passing `context.Context` from the frontend and mirror VarPool service usage.

## Tests / Verification
- Manual:
  - Open TopicBus page, add subscriptions, verify list persists across profile switch.
  - Publish events and confirm event list populates, filtering and detail view update correctly.
  - Apply max events and verify trimming works.

## Impact / Rollback
- Impact: New TopicBus page, state store, and storage keys; no server/core changes.
- Rollback: Revert the above files and remove the new change doc.
