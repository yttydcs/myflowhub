# Resource collections/actions QA evidence

This directory contains screenshots captured from the real Desktop UI. Browser-mode screenshots use the Wails application and production Go bindings; packaged screenshots use the freshly built `mfh-desktop.exe`. Both connected over loopback TCP to an isolated production Hub with real enrollment, policy, Flow, and filesystem providers. No mocked `DesktopAPI` or wire path was used.

The temporary fixture state, mount roots, AAR, APK, and packaged executable were removed after verification. Screenshots exclude the one-time permit, identity material, physical mount roots, and unrelated file contents.

## Browser-mode evidence

- `browser-resource-tree.png`: connected catalog with exactly the two Flow Collections and three filesystem Collections under descriptor-derived paths.
- `browser-pointer-context-menu.png`: pointer-opened menu on the real `storage/allowed` Resource.
- `browser-keyboard-context-menu.png`: keyboard-opened menu with readable view/add and descriptor-derived operations.
- `browser-collection-first-page.png`: real filesystem Collection page and bounded next-page control.
- `browser-filesystem-json-safe.png`: decoded JSON body visibly contains `fixture`, `<safe>`, and `items`; wrapper fields are not rendered as the inner value.
- `browser-safe-content-fallbacks.png`: HTML-like bytes remain literal text and the UI warns that HTML is not executed.
- `browser-filesystem-png-preview.png`: allowlisted PNG renderer path.
- `browser-filesystem-svg-safe-fallback.png`: SVG is explicitly not embedded or executed.
- `browser-filesystem-binary-safe-fallback.png`: unsupported binary content uses the non-inline fallback.
- `browser-forbidden-draft-retained.png`: authoritative Forbidden response with the entered key and max-bytes draft retained.
- `browser-flow-create-result.png`: explicit Flow create response with the typed schema and revision.
- `browser-flow-run-succeeded.png`: real Flow run Collection member in the `succeeded` state.
- `browser-light-saved-view.png` and `browser-dark-saved-view.png`: the same saved Collection View in both themes.

Live state assertions additionally confirmed:

- `ContextMenu` and `Shift+F10` open the menu; `Escape` closes it and restores focus to the triggering treeitem; the pure `storage` namespace has no resource menu.
- Opening or selecting a menu action performs no operation. Only the explicit Execute button sends the request.
- Both action cleanup paths pass: Return removes the focused action, while Close plus a later View/select does not resurrect draft/action state.
- Filesystem pagination advances from 64 to 76 members, removes the final next-page control, and keeps safe revision `r1`; nested directory enter/back also passes.
- The JSON renderer has no `script`, `iframe`, `object`, or `embed`; `fixture` and `<safe>` each occur once, while wrapper `Version` is absent.
- The PNG path contains one `img` with `alt=tiny.png`; the SVG and binary paths contain no executable/embed elements.
- Flow definitions Collection list/get returns the created `mfh.flow.definition.v1`; the explicit run appears through `flow/runs` and reaches `succeeded`.
- Forbidden is produced by the missing exact `read` grant, not by a local authorization guess, and the draft survives the server error.

## Packaged Desktop evidence

- `packaged-restored-view-light.jpg`: fresh packaged process automatically connects to the same Profile and restores the saved one-widget Collection View.
- `packaged-pointer-context-menu.jpg`: pointer context menu in the packaged window.
- `packaged-escaped-text-safe.jpg`: packaged filesystem read shows literal HTML-like text and the no-execution warning.
- `packaged-forbidden-draft-retained.jpg`: packaged authoritative Forbidden response with retained draft.
- `packaged-dark-forbidden.jpg`: the packaged window in dark theme.
- `packaged-second-restart-restored-dark.jpg`: after closing and starting the same executable again, the dark theme and one-widget View restore while the transient Inspector action, draft, and result do not.

The persisted `views.json` was 508 bytes with one View and one widget. A case-insensitive scan found zero member keys, tested file names/payloads, drafts, results, credentials, permits, permissions, `content` fields, or `key` fields.

## Final gate summary

- Frontend: 16 files / 128 tests passed; production build passed.
- Windows: clean Wails `windows/amd64` package passed and the fresh executable was used for both packaged launches.
- Generated contracts: canonical check passed with a temporary Git index; the real index hash and status were unchanged.
- Android: fresh AAR contains `arm64-v8a` and `x86_64` JNI libraries plus 11 generated Java binding classes. Offline JDK 21 Gradle `testDebugUnitTest`, `lintDebug`, and `assembleDebug` passed (53 tasks). No Android device was attached, so device smoke is recorded as unavailable rather than passed.
- Go: full tests, focused race tests, full vet, and `git diff --check` passed.
