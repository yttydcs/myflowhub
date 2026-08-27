package com.myflowhub.metricsnode

import kotlinx.coroutines.flow.MutableStateFlow

internal data class UiNodeState(
    val running: Boolean = false,
    val connection: String = "stopped",
    val statusJSON: String = "",
    val identityJSON: String = "",
    val error: String = "",
)

internal object NodeStateStore {
    val state = MutableStateFlow(UiNodeState())

    fun status(raw: String) {
        val connection = runCatching { org.json.JSONObject(raw).optString("state", "unknown") }.getOrDefault("unknown")
        state.value = state.value.copy(running = true, connection = connection, statusJSON = raw, error = "")
    }

    fun stopped() { state.value = UiNodeState() }
    fun identity(raw: String) { state.value = state.value.copy(identityJSON = raw, error = "") }
    fun error(error: Throwable) { state.value = state.value.copy(error = error.message ?: error.toString()) }
}
