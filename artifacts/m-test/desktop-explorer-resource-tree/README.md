# Desktop Explorer Resource Tree - m-test evidence

## Environment

- Packaged application: `apps/desktop/build/bin/myflowhub-desktop.exe`
- SHA-256: `954D84C0BEDA843E1E6F01FE271CB8241CA19A157D2F4E5EB20538428182B456`
- Hub endpoint: local `127.0.0.1:7331`
- View under test: existing four-widget View; no test widget was saved

## Evidence

1. [Resource path tree expanded](01-resource-tree-expanded.jpg)
   - `system` is a namespace branch.
   - `system/config` is both an actionable Resource and a parent.
   - `system/config/update` is visible as its child.
2. [Node collapsed, Resource full height](02-node-collapsed-resource-full-height.jpg)
   - Node content and separator are removed from the focus/layout path.
   - Resource receives the full Explorer height.
3. [Resource collapsed, Node full height](03-resource-collapsed-node-full-height.jpg)
   - Resource content and separator are removed from the focus/layout path.
   - Node receives the full Explorer height.

## Result

- Pointer disclosures and WAI-ARIA tree `ArrowLeft` behavior passed in the packaged application.
- The fixed Profile/connection footer remained visible.
- After validation, both sections, collapsed Resource branches, the light theme and the original four-widget View were restored.

