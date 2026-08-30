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
        val value = runCatching { org.json.JSONObject(raw) }.getOrNull()
        val connection = value?.optString("state", "unknown") ?: "unknown"
        val running = value?.optBoolean("running", false) ?: false
        state.value = state.value.copy(running = running, connection = connection, statusJSON = raw, error = "")
    }

    fun stopped() { state.value = UiNodeState() }
    fun identity(raw: String) { state.value = state.value.copy(identityJSON = raw, error = "") }
    fun error(error: Throwable) { state.value = state.value.copy(error = error.message ?: error.toString()) }
}
