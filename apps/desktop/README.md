# MyFlowHub Desktop

Canonical Wails desktop client for the MyFlowHub authority tree. The UI consumes the resource catalog and uses Variable snapshots, durable Stream subscriptions, and Commands through `sdk/bindings/desktop`.

See `docs/features/desktop.md` for architecture, storage, permissions, build, and migration decisions.

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
