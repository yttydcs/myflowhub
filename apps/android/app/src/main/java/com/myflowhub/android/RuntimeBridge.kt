package com.myflowhub.android

import android.content.Context
import android.net.Uri
import com.myflowhub.mobile.androidbinding.Androidbinding
import com.myflowhub.mobile.androidbinding.Client
import com.myflowhub.mobile.androidbinding.Host
import com.myflowhub.mobile.androidbinding.Listener
import java.io.File
import java.util.UUID
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import org.json.JSONObject

object RuntimeBridge {
    private val mutableState = MutableStateFlow(RuntimeState())
    val state: StateFlow<RuntimeState> = mutableState
    private var client: Client? = null
    private var host: Host? = null
    private var subscriptionId: Long = 0

    @Synchronized
    fun install(context: Context) {
        Androidbinding.setRFCOMMProvider(BluetoothProvider(context.applicationContext))
    }

    @Synchronized
    fun startClient(context: Context, settings: SettingsV1) {
        stopLocked()
        SettingsStore.validate(settings)
        val value = Androidbinding.newClient(File(context.filesDir, "client-vnext").absolutePath, settings.nodeId)
        try {
            value.trustParent(settings.parentId, settings.parentPublicKey)
            if (settings.transport == "rfcomm") value.startRFCOMM(settings.endpoint, settings.parentId, settings.permitJson)
            else value.startTCP(settings.endpoint, settings.parentId, settings.permitJson)
            client = value
            mutableState.value = RuntimeState(running = true, mode = "client", connection = "connecting", identityJson = value.identityJSON(), statusJson = value.statusJSON())
        } catch (error: Throwable) {
            runCatching { value.close() }
            fail(error)
            throw error
        }
    }

    @Synchronized
    fun startHost(context: Context, settings: SettingsV1) {
        stopLocked()
        SettingsStore.validate(settings)
        val value = Androidbinding.newHost(File(context.filesDir, "hub-vnext").absolutePath, settings.nodeId)
        try {
            if (settings.parentPublicKey.isNotBlank()) value.trustParent(settings.parentId, settings.parentPublicKey)
            value.setJoinPermit(settings.permitJson)
            value.start(settings.tcpListen, settings.rfcommListen)
            if (settings.endpoint.isNotBlank()) {
                if (settings.transport == "rfcomm") value.connectParentRFCOMM(settings.endpoint, settings.parentId)
                else value.connectParentTCP(settings.endpoint, settings.parentId)
            }
            host = value
            mutableState.value = RuntimeState(running = true, mode = "host", connection = "starting", identityJson = value.identityJSON(), statusJson = value.statusJSON())
        } catch (error: Throwable) {
            runCatching { value.stop() }
            fail(error)
            throw error
        }
    }

    @Synchronized
    fun refreshStatus() {
        try {
            val raw = client?.statusJSON() ?: host?.statusJSON() ?: "{\"running\":false,\"state\":\"disconnected\"}"
            val json = JSONObject(raw)
            val connection = if (json.has("parent")) json.getJSONObject("parent").optString("state", "disconnected") else json.optString("state", "disconnected")
            mutableState.value = mutableState.value.copy(statusJson = pretty(raw), connection = connection, error = "")
        } catch (error: Throwable) {
            fail(error)
        }
    }

    @Synchronized
    fun catalog(ownerNodeId: Long) {
        execute { it.catalogJSON(ownerNodeId, 15_000) }.also { mutableState.value = mutableState.value.copy(catalogJson = pretty(it), resultJson = pretty(it), error = "") }
    }

    @Synchronized
    fun snapshot(ownerNodeId: Long, name: String) {
        val raw = execute { it.snapshotJSON(ownerNodeId, name, 15_000) }
        mutableState.value = mutableState.value.copy(resultJson = pretty(raw), error = "")
    }

    @Synchronized
    fun invoke(ownerNodeId: Long, name: String, requestJson: String) {
        JSONObject(requestJson)
        val raw = execute { it.invokeJSON(ownerNodeId, name, requestJson, 15_000) }
        mutableState.value = mutableState.value.copy(resultJson = pretty(raw), error = "")
    }

    @Synchronized
    fun operate(ownerNodeId: Long, name: String, capability: String, schema: String, requestJson: String) {
        JSONObject(requestJson)
        val raw = execute { it.operateJSON(ownerNodeId, name, capability, schema, requestJson, 15_000) }
        mutableState.value = mutableState.value.copy(resultJson = pretty(raw), error = "")
    }

    @Synchronized
    fun subscribe(ownerNodeId: Long, name: String) {
        val value = requireClient()
        if (subscriptionId > 0) value.cancelSubscription(subscriptionId)
        subscriptionId = value.subscribe(ownerNodeId, name, 60_000, object : Listener {
            override fun onEvent(eventJSON: String) {
                mutableState.value = mutableState.value.copy(lastEventJson = pretty(eventJSON), error = "")
            }
            override fun onError(errorJSON: String) {
                mutableState.value = mutableState.value.copy(error = errorJSON)
            }
        })
    }

    @Synchronized
    fun upload(context: Context, uri: Uri, ownerNodeId: Long, destination: String, contentType: String) {
        val temporary = File(context.cacheDir, "upload-${UUID.randomUUID()}.bin")
        try {
            context.contentResolver.openInputStream(uri)?.use { input -> temporary.outputStream().use(input::copyTo) }
                ?: throw IllegalArgumentException("selected document cannot be opened")
            val raw = execute { it.uploadFile(ownerNodeId, temporary.absolutePath, destination, contentType, 10 * 60_000) }
            mutableState.value = mutableState.value.copy(resultJson = pretty(raw), error = "")
        } finally {
            temporary.delete()
        }
    }

    @Synchronized
    fun stop() = stopLocked()

    fun reportError(error: Throwable) = fail(error)

    private fun execute(block: (Client) -> String): String = try {
        block(requireClient())
    } catch (error: Throwable) {
        fail(error)
        throw error
    }

    private fun requireClient(): Client = client ?: throw IllegalStateException("resource operations require client mode")

    private fun stopLocked() {
        if (subscriptionId > 0) runCatching { client?.cancelSubscription(subscriptionId) }
        subscriptionId = 0
        runCatching { client?.close() }
        runCatching { host?.stop() }
        client = null
        host = null
        mutableState.value = RuntimeState()
    }

    private fun fail(error: Throwable) {
        mutableState.value = mutableState.value.copy(error = error.message ?: error.toString())
    }

    private fun pretty(raw: String): String = runCatching { JSONObject(raw).toString(2) }.getOrDefault(raw)
}
