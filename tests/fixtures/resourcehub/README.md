# Resource Hub GUI fixture

This test-only command starts a real persistent MyFlowHub Authority Hub and
registers three real read-only filesystem Collection Resources on that Hub
Node. It uses the production Hub, enrollment, policy, TCP and filesystem
implementations. It does not mock the wire protocol or Desktop API.

The caller owns all fixture directories. The command creates Hub state only
under the required absolute `-state` directory and never creates mount content.
Each mount root must already be an absolute real directory. The listener is
restricted to a loopback address; port `0` asks the OS for an available port.

Example:

```powershell
$env:GOWORK = 'off'
go run ./tests/fixtures/resourcehub `
  -state '<absolute-isolated-hub-state>' `
  -listen '127.0.0.1:0' `
  -id '<hub-node-id>' `
  -allowed-root '<absolute-allowed-root>' `
  -forbidden-root '<absolute-forbidden-root>' `
  -misc-root '<absolute-misc-root>'
```

Optional `-allowed-name`, `-forbidden-name`, `-misc-name` and matching `-label`
flags override the default `storage/allowed`, `storage/forbidden` and
`storage/misc` descriptors. The first stdout line is bounded startup JSON with
the actual endpoint and Resource names. It never contains physical roots,
permits or identity material.

## Admission and policy

Stop this fixture before using the canonical Hub command in an offline mode.
Both commands use the same `host/config` state and policy format, so identity,
Enrollment/Legacy Permit issuance and exact capability grants remain canonical:

```powershell
go run ./cmd/mfh-hub -id '<hub-node-id>' -state '<same-state>' -identity
go run ./cmd/mfh-hub -id '<hub-node-id>' -state '<same-state>' -issue-public-key '<device-public-key>' -issue-target-id '<hub-node-id>' -issue-role device
go run ./cmd/mfh-hub -id '<hub-node-id>' -state '<same-state>' -policy grant -subject '<desktop-node-id>' -action read -resource-node '<hub-node-id>' -resource system/catalog
```

Repeat the offline policy command for the exact filesystem
capabilities under test. A real Forbidden case should be created by omitting or
revoking one exact grant, not by changing this fixture or mocking the frontend.

Terminate the process with Ctrl+C or a normal process termination signal. The
filesystem registration is removed first, then the File controller and
Hub Node are closed through their production lifecycle.
