# Embedded leaf SDK

## Product boundary

Embedded is a constrained **leaf/client profile**, not a reduced Hub. C, MicroPython, and ESP32 use the same MFH4 v2 envelope, signed admission, Resource ID, Subscription, generic resource events, and operations as Go clients. They do not implement authority, routing, policy storage, or child admission.

## Implementations

### C99

- `embedded/c` is allocation-free. Callers own transport, frame buffers, identity persistence, randomness, Ed25519 operations, and queues.
- The default payload limit is 8 KiB. Metadata and payload lengths are checked before reading or encoding; the limit may only be raised at compile time together with caller-owned buffers.
- TCP, UART, Bluetooth, or another link may be supplied through exact-read/write callbacks. Link choice does not alter Resource or authority semantics.
- Identity uses a fixed versioned 112-byte representation. Invalid magic, version, size, IDs, phases, operations, resource names, or correlation rules fail explicitly.
- Join produces the canonical `join.v1` claim and requires the caller to verify the parent's signed `join-ack.v1`. Subscription and Command helpers never bypass Join.

### MicroPython

- `embedded/micropython` provides the same bounded codec and dependency-injected transport/crypto boundary.
- Identity is stored with version, Node ID, public key, and private key using a temporary file plus replace; a corrupt or mismatched identity is not regenerated silently.
- `Client` supports signed connect, subscribe/unsubscribe, Variable/Stream polling, Command calls, bounded retry backoff, and subscription recreation after reconnect.
- The package avoids CPython-only runtime dependencies; CPython unit tests exercise the MicroPython-compatible source.

### ESP32-S3 demo

- `embedded/esp32` is an ESP-IDF 6 project that directly consumes the C component.
- It uses Wi-Fi station + TCP, versioned NVS identity/config state, exact read/write loops, signed Join, parent signature verification, a `system/health` subscription, and a `notifications/publish` Command.
- Ed25519 uses the pinned Espressif-maintained `libsodium` component. ESP-IDF 6 broadly uses PSA Crypto, but the current Mbed TLS implementation does not implement Edwards curves; the demo therefore does not claim unsupported PSA Ed25519.
- A stored identity/config mismatch is fatal. Reset requires the explicit one-boot Kconfig switch. Production firmware should additionally use Secure Boot, Flash Encryption/NVS encryption, and an out-of-band credential provisioning path.

## Admission workflow

本节是迁移期兼容流程。C、MicroPython 与 ESP32 当前仍要求预置 Node ID 和父节点公钥，不发送 `MFHE` Enrollment；新的无 Node ID 首次注册协议见 [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)，其原生客户端迁移另行实施。

1. Configure a stable Node ID, parent ID/address, and parent public key.
2. Boot once to create and persist the Ed25519 identity; capture the logged public key.
3. Pre-trust the exact `(Node ID, public key)` or issue a one-use permit from the direct parent.
4. Configure the permit when needed, boot, and verify Join before any resource operation.
5. Reuse the persisted identity across restarts. Never generate a replacement after a decode or key mismatch error.

## Failure semantics

- Short reads/writes are retried by the platform adapter; EOF and socket errors are returned.
- Oversize frames, invalid metadata, invalid signatures, mismatched ack identity/nonce/epoch, unjoined operations, and unknown subscription events are errors.
- C keeps no hidden subscription queue. MicroPython keeps only subscription descriptors needed for reconnect.
- The ESP32 example stops on a permanent configuration or identity error and logs the failing phase without key, permit, or payload contents.

## Verification

- One complete MFH4 v2 operation frame fixture is shared by Go, C, and MicroPython and compared byte-for-byte.
- Host C tests build with C99 warnings as errors and cover codec, identity, Join, Subscription, Command, invalid/truncated input, and full golden parity.
- MicroPython tests cover codec limits, persistence mismatch, signed Join, Subscription, Command, reconnect restoration, and unjoined rejection.
- ESP-IDF build and real ESP32 flash/runtime smoke are environment-dependent gates. On the 2026-08-27 execution host, `idf.py` and a board were unavailable, so source is implemented but those two checks are not reported as passed.

## Non-goals

- Embedded Hub, router, authority, File, Flow, QUIC, and RFCOMM implementations.
- Compatibility with HeaderTcp, VarStore, TopicBus, SubProto, or the legacy wire.
- Silent fallback to generated identities, allow-all policy, or unsigned admission.
