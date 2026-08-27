# MyFlowHub MicroPython

Pure-Python bounded MFH3 leaf client. Platform code supplies a transport with `connect/read_exact/write_all/close` and a crypto provider with `generate_key/random/sign/verify`.

Identity is persisted as versioned JSON through an atomic temporary-file rename. The client verifies the trusted parent join acknowledgement, exposes catalog/Variable/Stream subscriptions and Command invocation, and recreates declared subscriptions after explicit reconnect.

The default payload ceiling is 8 KiB. This profile intentionally does not act as a Hub or authority.
