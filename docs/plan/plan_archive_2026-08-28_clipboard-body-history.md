# Plan - Clipboard body history

> 归档说明：该计划来自已退役的独立 ClipboardNode 仓库工作流。canonical monorepo 收口时保留为历史执行证据，不再作为根级当前状态入口。

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `feat/clipboard-body-history`
- Base: `master` at `92c01b1`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-body-history/MyFlowHub-ClipboardNode`
- Current Stage: 3.1 confirmed, entering 3.2
- Owner: main agent

## Stage Records

### Initialization

- `guide.md`: not present.
- Base/worktree confirmation: dedicated branch and worktree exist under `D:/project/MyFlowHub3/worktrees/`; implementation must stay in this worktree.
- Participating modules: `core/runtime`, `bridge`, `cmd/clipboardnode-bridge`, Flutter `app/lib/core/bridge`, Flutter shell UI, docs, tests.

### Stage 1 - Requirements Analysis

#### Goal

Add local clipboard text-body history to ClipboardNode. History means the clipboard body text itself, not only transfer metadata. The user must be able to configure how many body history entries are retained, and the default retention length is 256 entries.

#### Scope

Must:
- Store/display local in-memory body history in the UI when body history retention is enabled.
- Default retention mode to body history and default history length to 256.
- Keep status, logs/activity diagnostics, transfer manifests, and persistent config free of clipboard bodies except for the explicit local body-history surface.
- Add validation for history length and reject invalid settings explicitly.
- Update stable requirements/specs because previous docs described body retention as opt-in/off by default.
- Update Go and Flutter tests for defaults, contracts, state, and UI.

Optional:
- Preserve metadata-only and no-history modes for privacy controls.

Not doing:
- No persistent full clipboard history.
- No server-side storage or replay.
- No MyFlowHub protocol/subprotocol changes.
- No backward compatibility requirement for the previous default retention behavior.

#### Use Cases

1. A user sends or receives a text clipboard event and sees the actual text body in the History section.
2. A user changes the history length and the UI keeps only the newest configured number of body entries.
3. A user switches to metadata-only or no-history mode and body text is not retained in local history.
4. Activity/log views continue to show metadata without clipboard body exposure.

#### Functional Requirements

- Runtime config must include `history_retention` with `none`, `metadata`, and `body`.
- Runtime config must include `history_limit`; missing or zero values normalize to 256.
- `history_limit` must be validated as positive and bounded to avoid accidental unbounded memory retention.
- Runtime decisions for successful text publish/apply/pending paths may carry the text body for local UI history.
- Bridge activity payloads may include body text only when normalized runtime config has `history_retention=body`.
- Flutter engine state must maintain a separate body history list from activity/log metadata.
- History UI must render body history entries, not activity entries.
- Clear recent/history must clear both activity metadata and body history in the UI.

#### Non-functional Requirements

- Privacy: clipboard body must not be logged, included in status, transfer events, or persistent config.
- Performance: history is bounded by count; no unbounded list growth.
- Maintainability: keep Go runtime, bridge contract, Flutter model, and UI responsibilities separated.
- UX: settings must make the body/metadata/none retention choice and history length easy to see and edit.

#### Inputs / Outputs

Inputs:
- Local clipboard text.
- Remote TopicBus text events.
- Runtime/UI settings: `history_retention`, `history_limit`.

Outputs:
- UI body history entries with text, kind, device label, byte size, hash prefix, and timestamp.
- Activity/log metadata without body text.
- Normalized status/settings carrying retention mode and length but no body text.

#### Edge Cases

- Empty or invalid history length is rejected through existing settings validation paths.
- Changing the limit trims the current in-memory history immediately.
- Metadata-only and none modes keep body history empty.
- Oversize transfer manifests must not add body history because they do not carry inline body text through the bridge.
- Mobile native bindings may not return body text for remote decisions; local fallback can still record manual send text where available.

#### Acceptance Criteria

- Default settings show body history enabled with length 256.
- Sending text in preview adds the actual text to History.
- Live/web bridge activity events add body history only when activity JSON includes text and retention is body.
- Go status and transfer event tests still prove clipboard body is not leaked outside the explicit activity/history channel.
- Focused Go and Flutter tests pass.

#### Risks

- Body history increases local privacy sensitivity; UI and docs must be explicit and bounded.
- Existing tests that assumed metadata-only history need adjustment.
- Bridge events become conditionally sensitive; retention gating must be enforced on the Go side as well as UI side.

#### Issue List

- None.

### Stage 2 - Architecture Design

#### Overall Solution

Introduce `body` as the default history retention mode with a bounded `history_limit` defaulting to 256. Runtime `Decision` gains an optional `Text` field populated only for successful inline text paths. The bridge contract gains `history_limit` in settings/status and optional `text` in activity. The bridge emits `text` only when the normalized config requests body history. Flutter keeps body history as a distinct state list, renders it in the History section, and leaves logs/activity metadata as metadata-only.

#### Alternatives Considered

- UI-only local history from manual send text:
  - Rejected because live remote apply/pending events need body history too.
- Include text in status:
  - Rejected because status is diagnostic state and must remain body-free.
- Persist body history:
  - Rejected because the request only needs local bounded UI history and existing privacy constraints exclude persistent full history.

#### Module Responsibilities

- `core/runtime/config.go`: retention constants, default history limit, validation.
- `core/runtime/runtime.go`: populate optional decision text for inline text success paths.
- `bridge/contract.go`: JSON contract fields for `history_limit` and optional activity text.
- `cmd/clipboardnode-bridge/main.go`: map settings/status/history limit and gate body text emission by retention mode.
- Flutter bridge contract/state: retention enum, history limit, body history entry model.
- Flutter live/web/mobile/preview bridges: parse/apply status/settings, append bounded body history where available, clear and trim state.
- Flutter shell UI: render body history; add history retention and length controls.
- Docs/tests: update stable requirements/specs and validation coverage.

#### Data / Call Flow

Local send:
1. UI sends text to bridge.
2. Runtime validates, publishes, and returns a `Decision` with metadata and text.
3. Bridge emits activity metadata and includes `text` only when body history retention is enabled.
4. Flutter appends an in-memory body history entry and trims to `history_limit`.

Remote receive:
1. Runtime parses remote inline text.
2. If auto-apply is off, it records pending and emits a pending decision with text.
3. If auto-apply is on or user applies pending, it writes clipboard and emits an applied decision with text.
4. Flutter records body history only from explicit text-bearing activity payloads.

#### Interface Drafts

```go
const (
    HistoryRetentionNone     = "none"
    HistoryRetentionMetadata = "metadata"
    HistoryRetentionBody     = "body"
    DefaultHistoryLimit      = 256
)

type Config struct {
    HistoryRetention string `json:"history_retention,omitempty"`
    HistoryLimit     int    `json:"history_limit,omitempty"`
}

type Decision struct {
    Action Action
    EventID string
    Size int
    HashPrefix string
    Text string
}
```

```dart
enum HistoryRetention { none, metadata, body }

class ClipboardHistoryEntry {
  final String id;
  final ActivityKind kind;
  final String text;
  final int byteSize;
  final String hashPrefix;
  final DateTime timestamp;
}
```

#### Error Handling and Safety

- Invalid `history_retention` fails explicitly.
- Invalid `history_limit` fails explicitly; zero normalizes to the default.
- Body text is not emitted for transfer, validation, transport, ignored, or disabled decisions.
- Status and transfer JSON remain body-free by contract tests.

#### Performance and Testing Strategy

- History trimming is O(limit) on append using bounded list construction.
- Go tests:
  - runtime config defaults/validation
  - decision text propagation for local/remote/pending/apply paths
  - bridge status/body contracts
- Flutter tests:
  - default history controls show 256/body
  - preview send records body history
  - history limit trims displayed entries
  - log remains metadata-oriented

#### Extensibility Design Points

- Retention modes stay explicit so future persistent local history can be added behind a separate storage boundary.
- `history_limit` is shared across bridge and UI so live/web/preview behavior remains consistent.
- Body history remains UI state and not a runtime persistent store.

#### Issue List

- None.

### Stage 3.1 - Planning

#### Project Goal and Current State

The previous UI added separate History and Log navigation, but History still rendered activity metadata. This workflow changes History into a real clipboard body history view with configurable bounded length.

#### Docs Governance Routing Decision

Using `$m-docs`: stable behavior changed, so stable truth belongs in requirements/specs, while workflow execution belongs in root `plan.md` and final results in `docs/change`.

- Requirements impact: clarify
- Specs impact: clarify
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: none currently applicable

#### Executable Task List

##### T1 - Runtime retention config and decision text
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-body-history/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: Add `body` retention, default `history_limit=256`, validation, and optional text in successful inline decisions.
- Files / Modules: `core/runtime/config.go`, `core/runtime/runtime.go`, `core/runtime/*_test.go`
- Write Set: `core/runtime`
- Acceptance: Defaults normalize to body/256; invalid limits fail; local/remote/pending/apply decisions carry text only for inline success paths.
- Test Points: `go test ./core/runtime -count=1`
- Rollback: Revert `core/runtime` changes.

##### T2 - Bridge contract and privacy gating
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-body-history/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: Carry history settings through bridge and emit optional activity text only when body retention is enabled.
- Files / Modules: `bridge/contract.go`, `bridge/contract_test.go`, `cmd/clipboardnode-bridge/main.go`
- Write Set: `bridge`, `cmd/clipboardnode-bridge`
- Acceptance: Settings/status expose `history_limit`; status/transfer remain body-free; activity may include text for body history.
- Test Points: `go test ./bridge ./cmd/clipboardnode-bridge -count=1`
- Rollback: Revert bridge/cmd changes.

##### T3 - Flutter body history state and UI
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-body-history/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: Add body history model, bounded append behavior, settings length field, and actual text rendering in History.
- Files / Modules: `app/lib/core/bridge/*.dart`, `app/lib/features/shell/clipboard_shell.dart`, `app/test/widget_test.dart`
- Write Set: `app/lib`, `app/test`
- Acceptance: History shows text bodies, Settings shows retention and 256 length default, clearing clears history and logs.
- Test Points: `flutter test`; `flutter analyze`
- Rollback: Revert Flutter app changes.

##### T4 - Docs, validation, archive
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-body-history/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: Update stable docs, run focused/full validation, and archive in `docs/change`.
- Files / Modules: `docs/requirements/clipboard-sync.md`, `docs/specs/clipboard-sync.md`, `docs/change`
- Write Set: `docs`
- Acceptance: Requirements/specs describe body history default/limit/privacy boundary; change archive records tasks, tests, rollback, and lessons impact.
- Test Points: `git diff --check`; Go and Flutter validation listed above.
- Rollback: Revert docs/archive changes.

#### Dependencies

- T2 depends on T1 `Decision.Text` and config constants.
- T3 depends on T2 JSON contract for live/web body history.
- T4 depends on T1-T3 results and validation output.

#### Risks and Notes

- Keep body history in memory only.
- Do not edit main repo generated Flutter plugin files.
- Stop/relaunch any existing desktop app only during validation/preview.

#### Parallelism Assessment

No sub-agent split. T1-T3 share one sensitive privacy contract, and the write sets are sequentially dependent. Main agent retains implementation and review.

#### Issue List

- None.

阻塞：否
进入 3.2
