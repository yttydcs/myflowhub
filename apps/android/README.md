# MyFlowHub Android

Canonical Android client and optional local Hub host. The app directly consumes the gomobile types generated from `sdk/bindings/android`; no reflection or sibling modules are used.

The product supports TCP and Bluetooth Classic RFCOMM through the same identity, parent supervision, catalog, subscription, Command and File APIs. Runtime Bluetooth permission, disabled-adapter, and unsupported-device failures are explicit.

The AAR is generated and intentionally not committed:

```powershell
$env:GOWORK='off'
gomobile bind '-target=android/arm64,android/amd64' -androidapi 26 -javapkg com.myflowhub.mobile -o apps/android/app/libs/myflowhub.aar ./sdk/bindings/android
```

Then run `apps/android/gradlew testDebugUnitTest assembleDebug lintDebug`.
