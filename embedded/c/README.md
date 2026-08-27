# MyFlowHub Embedded C

C99 bounded leaf/client profile for the MFH3 v1 envelope. The library performs no allocation: callers own frame buffers, identity storage, transport I/O, randomness, Ed25519 signing, and join-ack verification.

The default embedded payload limit is 8 KiB and queues are not hidden inside the library. File chunks and other larger resource values require an explicitly larger compile-time `MFH_MAX_PAYLOAD` and corresponding buffers.

Build host tests with CMake/CTest. ESP-IDF consumes this directory as a component.
