# Desktop nested docking m-test evidence

- Date: 2026-08-30
- Worktree: `desktop-nested-docking`
- Branch: `codex/desktop-nested-docking`
- Test config: isolated clone under the current user's temporary directory; removed during archive cleanup.
- Real service: workflow-owned local `mfh-hub` on `127.0.0.1:7331`; stopped after testing.
- Final package: `apps/desktop/build/bin/myflowhub-desktop.exe`
- Final SHA-256: `3C94D2DB1B474D43D03B0529B472BA41D39CF9E23313CC17F0504C59269DE560`

## Results

| Area | Result | Evidence |
| --- | --- | --- |
| v2 migration | Passed | Four legacy widgets migrated without loss; first v3 save created `views.pre-v3.json` with version 2 and wrote `views.json` version 3. |
| Resize | Passed | Real pointer resize changed adjacent weights and dirtied the View; keyboard Right adjusted the focused separator; component tests prove RAF preview and release-only commit. |
| Drag docking | Passed | Real widget drag produced a vertical root with a full-width top pane and a three-column lower split; pure domain tests cover exact `A | C | B`, `A | (B / C)` and `(A | B | C) / D`. |
| 6–8 panes | Passed | Six-pane 2×3 migration rendered in light/dark themes; adding two panes at 986 px preserved minimums and exposed a working horizontal scrollbar. |
| Persistence | Passed | Save, reopen and two packaged-process restarts restored the nested topology; the final production package also restored the dark preference on restart. |
| Error states | Passed | A detached resource retained its pane with `资源暂不可用`; a real `file/transfers` permission denial stayed explicit in Inspector and Widget. |
| Automated | Passed | Vitest 8 files / 44 tests; `go test ./...`; TypeScript/Vite build; Wails Windows production build. |

## Screenshots

- `01-migrated-four-pane-light.jpg`: migrated four-pane View with real catalog data.
- `03-keyboard-resize-four-pane.jpg`: focused accessible separator after pointer/keyboard adjustment.
- `04-dragged-root-top-nested-layout.jpg`: drag-created vertical root and nested lower row.
- `05-six-pane-light.jpg` / `06-six-pane-dark.jpg`: six-pane theme coverage.
- `07-restart-persistence-dark.jpg`: saved topology after restart.
- `08-six-pane-narrow-dark.jpg`: 986 px six-pane layout.
- `09-eight-pane-narrow-scroll.jpg` / `10-eight-pane-horizontal-scroll.jpg`: eight panes before and after actual horizontal scrolling.
- `11-final-packaged-build-smoke.jpg` / `12-final-build-restart-dark.jpg`: final Wails artifact smoke and restart.
- `13-forbidden-inspector.jpg` / `14-forbidden-widget.jpg`: explicit real permission denial states.

No credential, permit, private key or Resource secret is present in these screenshots.
