package com.myflowhub.android

import android.content.Context
import org.json.JSONObject

class SettingsStore(context: Context) {
    private val preferences = context.getSharedPreferences("myflowhub-vnext", Context.MODE_PRIVATE)

    fun load(): SettingsV1 {
        val raw = preferences.getString(KEY, null) ?: return SettingsV1()
        return decode(raw)
    }

    fun save(value: SettingsV1) {
        validate(value)
        if (!preferences.edit().putString(KEY, encode(value)).commit()) {
            throw IllegalStateException("failed to persist Android settings")
        }
    }

    fun reset(confirmation: String): SettingsV1 {
        require(confirmation == "RESET ANDROID V1") { "explicit reset requires RESET ANDROID V1" }
        val value = SettingsV1()
        save(value)
        return value
    }

    companion object {
        private const val KEY = "settings.v1"

        fun encode(value: SettingsV1): String {
            validate(value)
            return JSONObject()
                .put("version", 1).put("mode", value.mode).put("node_id", value.nodeId)
                .put("parent_id", value.parentId).put("transport", value.transport)
                .put("endpoint", value.endpoint).put("parent_public_key", value.parentPublicKey)
                .put("permit_json", value.permitJson).put("tcp_listen", value.tcpListen)
                .put("rfcomm_listen", value.rfcommListen).toString()
        }

        fun decode(raw: String): SettingsV1 {
            val json = JSONObject(raw)
            require(json.optInt("version", 0) == 1) { "unsupported Android settings version; explicit reset required" }
            val allowed = setOf("version", "mode", "node_id", "parent_id", "transport", "endpoint", "parent_public_key", "permit_json", "tcp_listen", "rfcomm_listen")
            require(json.keys().asSequence().all { it in allowed }) { "unknown Android settings field; explicit reset required" }
            return SettingsV1(
                version = 1, mode = json.getString("mode"), nodeId = json.getLong("node_id"),
                parentId = json.getLong("parent_id"), transport = json.getString("transport"),
                endpoint = json.getString("endpoint"), parentPublicKey = json.getString("parent_public_key"),
                permitJson = json.getString("permit_json"), tcpListen = json.getString("tcp_listen"),
                rfcommListen = json.getString("rfcomm_listen"),
            ).also(::validate)
        }

        fun validate(value: SettingsV1) {
            require(value.version == 1) { "settings version must be 1" }
            require(value.mode == "client" || value.mode == "host") { "mode must be client or host" }
            require(value.transport == "tcp" || value.transport == "rfcomm") { "transport must be tcp or rfcomm" }
            require(value.nodeId > 0 && value.parentId > 0) { "NodeIDs must be positive" }
            require(value.endpoint.length <= 512 && value.parentPublicKey.length <= 512) { "connection setting is too large" }
            require(value.permitJson.length <= 1_048_576) { "permit is too large" }
            if (value.permitJson.isNotBlank()) JSONObject(value.permitJson)
            require(value.tcpListen.length <= 512 && value.rfcommListen.length <= 512) { "listener setting is too large" }
        }
    }
}
