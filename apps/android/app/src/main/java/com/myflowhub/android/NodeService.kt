package com.myflowhub.android

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.IBinder
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import java.util.concurrent.Executors

class NodeService : Service() {
    private val monitor = Executors.newSingleThreadExecutor()
    @Volatile private var stopped = false

    override fun onCreate() {
        super.onCreate()
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(NotificationChannel(CHANNEL, "MyFlowHub runtime", NotificationManager.IMPORTANCE_LOW))
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID,
            NotificationCompat.Builder(this, CHANNEL).setSmallIcon(android.R.drawable.stat_sys_upload).setContentTitle("MyFlowHub").setContentText("Runtime active").setOngoing(true).build(),
            if (Build.VERSION.SDK_INT >= 29) ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC else 0,
        )
        monitor.execute {
            while (!stopped) {
                RuntimeBridge.refreshStatus()
                Thread.sleep(2_000)
            }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            RuntimeBridge.stop()
            stopSelf()
            return START_NOT_STICKY
        }
        return START_STICKY
    }

    override fun onDestroy() {
        stopped = true
        monitor.shutdownNow()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    companion object {
        const val ACTION_KEEP = "com.myflowhub.android.KEEP"
        const val ACTION_STOP = "com.myflowhub.android.STOP"
        private const val CHANNEL = "myflowhub-runtime"
        private const val NOTIFICATION_ID = 1201
    }
}
