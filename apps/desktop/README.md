# MyFlowHub Desktop

Canonical Wails desktop client for the MyFlowHub authority tree. The UI consumes the resource catalog and uses Variable snapshots, durable Stream subscriptions, and Commands through `sdk/bindings/desktop`.

See `docs/features/desktop.md` for architecture, storage, permissions, build, and migration decisions.

## Brand assets

Desktop uses the accepted Coupled Seam V5 assets from `design-demos/brand/`:

- `frontend/public/brand/` contains the full, compact, and light/dark tray SVGs used by the React shell and favicon.
- `build/appicon.png` is the 1024px Wails app source derived from `myflowhub-app-icon-source-v5.svg`.
- `build/windows/icon.ico` contains the 16–256px Windows icon set derived from the same V5 app source.

Do not resize the full symbol for compact or tray use; select the corresponding V5 optical variant.

Quick validation:

```powershell
$env:GOWORK='off'
go test ./apps/desktop/...
cd apps/desktop
wails generate module
cd frontend
npm ci
npm test
npm run build
cd ..
wails build -platform windows/amd64
```
