package com.myflowhub.android

data class SettingsV1(
    val version: Int = 1,
    val mode: String = "client",
    val nodeId: Long = 2,
    val parentId: Long = 1,
    val transport: String = "tcp",
    val endpoint: String = "10.0.2.2:7341",
    val parentPublicKey: String = "",
    val permitJson: String = "",
    val tcpListen: String = "127.0.0.1:7341",
    val rfcommListen: String = "",
)

data class RuntimeState(
    val running: Boolean = false,
    val mode: String = "stopped",
    val connection: String = "disconnected",
    val identityJson: String = "",
    val statusJson: String = "{}",
    val catalogJson: String = "{}",
    val resultJson: String = "{}",
    val lastEventJson: String = "",
    val error: String = "",
)
