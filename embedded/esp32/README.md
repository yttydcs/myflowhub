# MyFlowHub ESP32 leaf demo

This ESP-IDF 6 project consumes `embedded/c` directly and runs the real MFH3 leaf profile over Wi-Fi/TCP. It persists a versioned Ed25519 identity and compiled connection profile in NVS, verifies the parent's signed join acknowledgement, subscribes to `system/health`, and calls `notifications/publish`.

Ed25519 is provided by Espressif's pinned `libsodium` component. ESP-IDF 6 uses PSA Crypto broadly, but its current Mbed TLS implementation does not implement the Edwards curve family, so the demo does not pretend that the PSA specification alone provides a working Ed25519 backend.

Configure with `idf.py menuconfig` under **MyFlowHub ESP32 demo**. Set Wi-Fi, parent address/ID/public key, node ID, and an optional one-use permit. On first boot the public key is logged for provisioning. A stored identity or config mismatch is an explicit error; enable reset for exactly one boot only when intentional. Production firmware should enable Secure Boot, Flash Encryption/NVS encryption, and provision credentials outside `sdkconfig`.

Build and flash:

```text
idf.py set-target esp32s3
idf.py build
idf.py flash monitor
```

The demo has no Hub, router, or authority implementation. Bluetooth/RFCOMM, QUIC, File, and Flow are deliberately outside the constrained leaf profile.
