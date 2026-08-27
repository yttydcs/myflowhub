package com.myflowhub.myflowhub_clipboard

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import com.myflowhub.gomobile.clipboardmobile.Clipboardmobile
import com.myflowhub.gomobile.clipboardmobile.Client
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import org.json.JSONObject
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean

class MainActivity : FlutterActivity() {
    private val executor = Executors.newSingleThreadExecutor()
    private val mainHandler = Handler(Looper.getMainLooper())
    private val active = AtomicBoolean(true)
    private lateinit var clipboard: ClipboardManager
    private lateinit var client: Client

    private val clipboardListener = ClipboardManager.OnPrimaryClipChangedListener {
        val text = clipboard.primaryClip
            ?.getItemAt(0)
            ?.coerceToText(this)
            ?.toString()
            .orEmpty()
        if (text.isNotEmpty()) {
            executor.execute {
                runCatching { client.observeText(text) }
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        client = Clipboardmobile.newClient()
        clipboard.addPrimaryClipChangedListener(clipboardListener)
        scheduleWrites()
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            "com.myflowhub.clipboard/bridge",
        ).setMethodCallHandler { call, result ->
            if (call.method != "call") {
                result.notImplemented()
                return@setMethodCallHandler
            }
            executor.execute {
                runCatching { dispatch(call) }
                    .onSuccess { response -> mainHandler.post { result.success(response) } }
                    .onFailure { error ->
                        mainHandler.post {
                            result.error(
                                "CLIPBOARD_BRIDGE",
                                error.message ?: "Clipboard bridge failed",
                                null,
                            )
                        }
                    }
            }
        }
    }

    private fun dispatch(call: MethodCall): String {
        val encoded = call.arguments as? String
            ?: error("Clipboard bridge request must be JSON")
        require(encoded.toByteArray(Charsets.UTF_8).size <= 512 * 1024) {
            "Clipboard bridge request exceeds 512 KiB"
        }
        val envelope = JSONObject(encoded)
        val operation = envelope.getString("op")
        val payload = envelope.getJSONObject("payload")
        normalizeStateDirectory(payload)
        return when (operation) {
            "identity" -> client.identity(payload.toString())
            "start" -> client.start(payload.toString())
            "stop" -> {
                client.stop()
                "{\"version\":1,\"stopped\":true}"
            }
            "status" -> client.status()
            "configuration" -> client.configuration()
            "configuration.update" -> client.updateConfiguration(payload.toString())
            "send" -> client.sendText(payload.getString("text"))
            "apply" -> client.applyPending(payload.getString("event_id"))
            "history" -> client.history()
            "history.clear" -> client.clearHistory()
            else -> error("Unsupported clipboard bridge operation")
        }
    }

    private fun normalizeStateDirectory(payload: JSONObject) {
        if (payload.optString("state_directory") == "<app-data>") {
            payload.put("state_directory", filesDir.resolve("clipboard").absolutePath)
        }
    }

    private fun scheduleWrites() {
        mainHandler.postDelayed(object : Runnable {
            override fun run() {
                if (!active.get()) return
                executor.execute {
                    val encoded = runCatching { client.nextWrite() }.getOrNull().orEmpty()
                    if (encoded.isNotEmpty()) {
                        val action = JSONObject(encoded)
                        val actionID = action.getString("action_id")
                        val text = action.getString("text")
                        mainHandler.post {
                            val failure = runCatching {
                                clipboard.setPrimaryClip(ClipData.newPlainText("MyFlowHub", text))
                            }.exceptionOrNull()
                            executor.execute {
                                runCatching {
                                    client.completeWrite(actionID, failure?.javaClass?.simpleName.orEmpty())
                                }
                            }
                        }
                    }
                }
                mainHandler.postDelayed(this, 100)
            }
        }, 100)
    }

    override fun onDestroy() {
        active.set(false)
        clipboard.removePrimaryClipChangedListener(clipboardListener)
        executor.execute { runCatching { client.stop() } }
        executor.shutdown()
        super.onDestroy()
    }
}
