package com.myflowhub.android

import android.Manifest
import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothManager
import android.bluetooth.BluetoothServerSocket
import android.bluetooth.BluetoothSocket
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import com.myflowhub.mobile.androidbinding.RFCOMMListener
import com.myflowhub.mobile.androidbinding.RFCOMMPipe
import com.myflowhub.mobile.androidbinding.RFCOMMProvider
import java.util.UUID

class BluetoothProvider(private val context: Context) : RFCOMMProvider {
    @SuppressLint("MissingPermission")
    override fun dial(bdaddr: String, uuid: String, channel: Long, secure: Boolean): RFCOMMPipe {
        requireBluetooth("dial RFCOMM")
        val adapter = adapter()
        adapter.cancelDiscovery()
        val device = adapter.getRemoteDevice(bdaddr)
        val service = UUID.fromString(uuid)
        val socket = if (secure) device.createRfcommSocketToServiceRecord(service) else device.createInsecureRfcommSocketToServiceRecord(service)
        try {
            socket.connect()
            return BluetoothPipe(socket)
        } catch (error: Throwable) {
            runCatching { socket.close() }
            throw IllegalStateException("RFCOMM dial failed for $bdaddr: ${error.message ?: error}", error)
        }
    }

    @SuppressLint("MissingPermission")
    override fun listen(uuid: String, secure: Boolean): RFCOMMListener {
        requireBluetooth("listen RFCOMM")
        val service = UUID.fromString(uuid)
        val socket = if (secure) adapter().listenUsingRfcommWithServiceRecord("MyFlowHub", service)
        else adapter().listenUsingInsecureRfcommWithServiceRecord("MyFlowHub", service)
        return BluetoothListener(socket, uuid)
    }

    private fun adapter(): BluetoothAdapter {
        val value = context.getSystemService(BluetoothManager::class.java)?.adapter
            ?: throw IllegalStateException("Bluetooth Classic is not available on this device")
        check(value.isEnabled) { "Bluetooth is disabled" }
        return value
    }

    private fun requireBluetooth(action: String) {
        if (Build.VERSION.SDK_INT >= 31 && context.checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) != PackageManager.PERMISSION_GRANTED) {
            throw SecurityException("BLUETOOTH_CONNECT permission is required to $action")
        }
    }
}

private class BluetoothListener(private val socket: BluetoothServerSocket, private val uuid: String) : RFCOMMListener {
    override fun accept(): RFCOMMPipe = try {
        BluetoothPipe(socket.accept())
    } catch (error: Throwable) {
        throw IllegalStateException("RFCOMM accept failed: ${error.message ?: error}", error)
    }
    override fun close() = socket.close()
    override fun addr(): String = "bt+rfcomm://listen?uuid=$uuid"
}

private class BluetoothPipe(private val socket: BluetoothSocket) : RFCOMMPipe {
    override fun read(data: ByteArray): Long = socket.inputStream.read(data).toLong()
    override fun write(data: ByteArray): Long {
        socket.outputStream.write(data)
        return data.size.toLong()
    }
    override fun close() = socket.close()
    override fun remoteBDAddr(): String = runCatching { socket.remoteDevice.address }.getOrDefault("")
}
