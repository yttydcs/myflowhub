package com.myflowhub.metricsnode

import android.app.ActivityManager
import android.content.Context
import android.hardware.camera2.CameraCharacteristics
import android.hardware.camera2.CameraManager
import android.media.AudioManager
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.BatteryManager
import android.provider.Settings
import java.io.RandomAccessFile
import kotlin.math.roundToInt

internal class MetricPlatform(private val context: Context) {
    private val audio = context.getSystemService(AudioManager::class.java)
    private val connectivity = context.getSystemService(ConnectivityManager::class.java)
    private val activity = context.getSystemService(ActivityManager::class.java)
    private val battery = context.getSystemService(BatteryManager::class.java)
    private val camera = context.getSystemService(CameraManager::class.java)
    private var previousCPU: Pair<Long, Long>? = null
    private var torchEnabled = false

    fun collect(metric: String): String = when (metric) {
        "battery_percent" -> battery.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY).coerceIn(0, 100).toString()
        "battery_charging" -> (battery.isCharging).toString()
        "battery_on_ac" -> battery.isCharging.toString()
        "network_online" -> (activeCapabilities() != null).toString()
        "network_type" -> networkType(activeCapabilities())
        "cpu_percent" -> cpuPercent().toString()
        "memory_percent" -> memoryPercent().toString()
        "volume_percent" -> volumePercent().toString()
        "volume_muted" -> audio.isStreamMute(AudioManager.STREAM_MUSIC).toString()
        "brightness_percent" -> brightnessPercent().toString()
        "flashlight_enabled" -> torchEnabled.toString()
        else -> error("unsupported Android metric: $metric")
    }

    fun apply(metric: String, value: String): String = when (metric) {
        "volume_percent" -> {
            val max = audio.getStreamMaxVolume(AudioManager.STREAM_MUSIC)
            audio.setStreamVolume(AudioManager.STREAM_MUSIC, Percent.value(value, max), 0)
            volumePercent().toString()
        }
        "volume_muted" -> {
            val enabled = value.toBooleanStrict()
            audio.adjustStreamVolume(
                AudioManager.STREAM_MUSIC,
                if (enabled) AudioManager.ADJUST_MUTE else AudioManager.ADJUST_UNMUTE,
                0,
            )
            audio.isStreamMute(AudioManager.STREAM_MUSIC).toString()
        }
        "brightness_percent" -> {
            check(Settings.System.canWrite(context)) { "WRITE_SETTINGS permission is required" }
            Settings.System.putInt(context.contentResolver, Settings.System.SCREEN_BRIGHTNESS, Percent.value(value, 255))
            brightnessPercent().toString()
        }
        "flashlight_enabled" -> {
            val enabled = value.toBooleanStrict()
            camera.setTorchMode(torchCameraID(), enabled)
            torchEnabled = enabled
            torchEnabled.toString()
        }
        else -> error("unsupported Android control: $metric")
    }

    private fun activeCapabilities(): NetworkCapabilities? {
        val network = connectivity.activeNetwork ?: return null
        return connectivity.getNetworkCapabilities(network)
    }

    private fun networkType(capabilities: NetworkCapabilities?): String = when {
        capabilities == null -> "offline"
        capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "wifi"
        capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "cellular"
        capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "ethernet"
        capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN) -> "vpn"
        else -> "other"
    }

    private fun memoryPercent(): Int {
        val info = ActivityManager.MemoryInfo()
        activity.getMemoryInfo(info)
        if (info.totalMem <= 0) return 0
        return (((info.totalMem - info.availMem).toDouble() / info.totalMem) * 100).roundToInt().coerceIn(0, 100)
    }

    private fun volumePercent(): Int {
        val max = audio.getStreamMaxVolume(AudioManager.STREAM_MUSIC).coerceAtLeast(1)
        return (audio.getStreamVolume(AudioManager.STREAM_MUSIC) * 100.0 / max).roundToInt().coerceIn(0, 100)
    }

    private fun brightnessPercent(): Int {
        val raw = Settings.System.getInt(context.contentResolver, Settings.System.SCREEN_BRIGHTNESS, 0)
        return (raw * 100.0 / 255).roundToInt().coerceIn(0, 100)
    }

    private fun cpuPercent(): Int {
        val fields = RandomAccessFile("/proc/stat", "r").use { it.readLine() }.trim().split(Regex("\\s+")).drop(1).map { it.toLong() }
        val idle = fields.getOrElse(3) { 0 } + fields.getOrElse(4) { 0 }
        val total = fields.sum()
        val previous = previousCPU
        previousCPU = total to idle
        if (previous == null || total <= previous.first) return 0
        return (((total - previous.first - (idle - previous.second)) * 100.0) / (total - previous.first)).roundToInt().coerceIn(0, 100)
    }

    private fun torchCameraID(): String = camera.cameraIdList.firstOrNull { id ->
        camera.getCameraCharacteristics(id).get(CameraCharacteristics.FLASH_INFO_AVAILABLE) == true
    } ?: error("device has no controllable flashlight")
}

internal object Percent {
    fun value(percent: String, maximum: Int): Int {
        val parsed = percent.toDoubleOrNull() ?: error("percent is not numeric")
        require(parsed in 0.0..100.0) { "percent must be between 0 and 100" }
        return (parsed * maximum / 100.0).roundToInt().coerceIn(0, maximum)
    }
}
