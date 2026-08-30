package com.myflowhub.metricsnode

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.util.Base64
import androidx.core.app.NotificationCompat
import org.json.JSONObject
import java.io.File

class NodeService : Service() {
    private var bridge: MobileBridge? = null
    private lateinit var platform: MetricPlatform
    private val runtimeGeneration = RuntimeGeneration()
    private var worker: Thread? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        platform = MetricPlatform(this)
        createChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> stopNode()
            ACTION_START -> startNode(intent)
            else -> restoreNode()
        }
        return START_STICKY
    }

    override fun onDestroy() {
        var current: MobileBridge? = null
        runtimeGeneration.invalidate {
            current = bridge
            bridge = null
            stopWorker()
        }
        runCatching { current?.stop() }
        super.onDestroy()
    }

    private fun startNode(intent: Intent) {
        startForeground(NOTIFICATION_ID, notification("Starting"))
        val request = JSONObject()
            .put("version", 1)
            .put("state_directory", File(filesDir, "metrics-vnext").absolutePath)
            .put("node_id", intent.getStringExtra(EXTRA_NODE_ID).orEmpty())
            .put("parent_node_id", intent.getStringExtra(EXTRA_PARENT_ID).orEmpty())
            .put("endpoint", intent.getStringExtra(EXTRA_ENDPOINT).orEmpty())
            .put("parent_public_key", intent.getStringExtra(EXTRA_PARENT_KEY).orEmpty())
        intent.getStringExtra(EXTRA_PERMIT)?.trim()?.takeIf { it.isNotEmpty() }?.let { request.put("permit", JSONObject(it)) }
        val raw = request.toString()
        getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putString(KEY_REQUEST, raw).putBoolean(KEY_DESIRED, true).apply()
        launch(raw)
    }

    private fun restoreNode() {
        val prefs = getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        if (!prefs.getBoolean(KEY_DESIRED, false)) {
            stopSelf()
            return
        }
        val raw = prefs.getString(KEY_REQUEST, null)
        if (raw == null) {
            prefs.edit().putBoolean(KEY_DESIRED, false).apply()
            stopSelf()
            return
        }
        startForeground(NOTIFICATION_ID, notification("Restoring"))
        launch(raw)
    }

    private fun launch(request: String) {
        var previous: MobileBridge? = null
        val generation = runtimeGeneration.begin {
            previous = bridge
            bridge = null
            stopWorker()
        }
        runCatching { previous?.stop() }.onFailure(NodeStateStore::error)
        Thread {
            try {
                val candidate = MobileBridge()
                val status = candidate.start(request)
                val installed = runtimeGeneration.commit(generation) {
                    bridge = candidate
                    NodeStateStore.status(status)
                    getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, notification("Running"))
                    startWorker(candidate, generation)
                }
                if (!installed) runCatching { candidate.stop() }
            } catch (error: Throwable) {
                if (runtimeGeneration.isCurrent(generation)) {
                    NodeStateStore.error(error)
                    getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, notification("Failed: ${error.message.orEmpty().take(80)}"))
                }
            }
        }.also { it.name = "metrics-runtime-launch-$generation"; it.start() }
    }

    private fun startWorker(current: MobileBridge, generation: Long) {
        stopWorker()
        worker = Thread {
            val nextDue = mutableMapOf<String, Long>()
            while (runtimeGeneration.isCurrent(generation) && !Thread.currentThread().isInterrupted) {
                try {
                    val now = System.currentTimeMillis()
                    val config = JSONObject(current.configuration())
                    val settings = config.getJSONArray("settings")
                    for (index in 0 until settings.length()) {
                        val setting = settings.getJSONObject(index)
                        val metric = setting.getString("metric")
                        if (!setting.getBoolean("enabled") || now < (nextDue[metric] ?: 0)) continue
                        val interval = setting.getLong("interval_ms")
                        runCatching { platform.collect(metric) }
                            .onSuccess { current.updateMetric(metric, it, "") }
                            .onFailure { current.updateMetric(metric, "", it.message ?: it.toString()) }
                        nextDue[metric] = now + interval
                    }
                    drainActions(current)
                    drainNotifications(current)
                    NodeStateStore.status(current.status())
                } catch (error: Throwable) {
                    if (!runtimeGeneration.isCurrent(generation) || Thread.currentThread().isInterrupted) return@Thread
                    NodeStateStore.error(error)
                }
                try {
                    Thread.sleep(250)
                } catch (_: InterruptedException) {
                    return@Thread
                }
            }
        }.also { it.name = "metrics-platform-worker"; it.start() }
    }

    private fun drainActions(current: MobileBridge) {
        repeat(8) {
            val raw = current.nextAction()
            if (raw.isBlank()) return
            val action = JSONObject(raw)
            val id = action.getString("action_id")
            runCatching { platform.apply(action.getString("metric"), action.getString("value")) }
                .onSuccess { current.completeAction(id, it, "") }
                .onFailure { current.completeAction(id, "", it.message ?: it.toString()) }
        }
    }

    private fun drainNotifications(current: MobileBridge) {
        repeat(8) {
            val raw = current.nextNotification()
            if (raw.isBlank()) return
            val event = JSONObject(raw)
            val attributes = event.optJSONObject("attributes")
            val title = attributes?.optString("title")?.takeIf { it.isNotBlank() }
                ?: "MyFlowHub / ${event.getString("channel")}"
            val contentType = event.getString("content_type")
            val body = if (contentType.startsWith("text/")) {
                String(Base64.decode(event.getString("body"), Base64.DEFAULT), Charsets.UTF_8).trim()
            } else {
                "New $contentType notification"
            }
            val notification = NotificationCompat.Builder(this, EVENT_CHANNEL)
                .setSmallIcon(android.R.drawable.stat_notify_more)
                .setContentTitle(title.take(96))
                .setContentText(body.take(220))
                .setStyle(NotificationCompat.BigTextStyle().bigText(body.take(220)))
                .setAutoCancel(true)
                .build()
            getSystemService(NotificationManager::class.java).notify(event.getString("event_id").hashCode(), notification)
        }
    }

    private fun stopNode() {
        getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().clear().apply()
        var current: MobileBridge? = null
        runtimeGeneration.invalidate {
            current = bridge
            bridge = null
            stopWorker()
        }
        runCatching { current?.stop() }.onFailure(NodeStateStore::error)
        NodeStateStore.stopped()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun stopWorker() {
        worker?.interrupt()
        worker = null
    }

    private fun createChannel() {
        getSystemService(NotificationManager::class.java).createNotificationChannel(
            NotificationChannel(CHANNEL, "Metrics node", NotificationManager.IMPORTANCE_LOW),
        )
        getSystemService(NotificationManager::class.java).createNotificationChannel(
            NotificationChannel(EVENT_CHANNEL, "MyFlowHub notifications", NotificationManager.IMPORTANCE_DEFAULT),
        )
    }

    private fun notification(text: String) = NotificationCompat.Builder(this, CHANNEL)
        .setSmallIcon(android.R.drawable.stat_notify_sync)
        .setContentTitle("MyFlowHub Metrics")
        .setContentText(text)
        .setOngoing(true)
        .build()

    companion object {
        const val ACTION_START = "com.myflowhub.metricsnode.START"
        const val ACTION_STOP = "com.myflowhub.metricsnode.STOP"
        const val EXTRA_NODE_ID = "node_id"
        const val EXTRA_PARENT_ID = "parent_id"
        const val EXTRA_ENDPOINT = "endpoint"
        const val EXTRA_PARENT_KEY = "parent_key"
        const val EXTRA_PERMIT = "permit"
        private const val PREFS = "metrics-vnext"
        private const val KEY_REQUEST = "start_request"
        private const val KEY_DESIRED = "desired"
        private const val CHANNEL = "metrics-runtime"
        private const val EVENT_CHANNEL = "myflowhub-events"
        private const val NOTIFICATION_ID = 4101
    }
}
