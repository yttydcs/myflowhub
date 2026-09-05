package com.myflowhub.android

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import java.util.concurrent.atomic.AtomicReference
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    private val permissions = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { }
    private val selectedDocument = AtomicReference<Uri?>(null)
    private val documentPicker = registerForActivityResult(ActivityResultContracts.OpenDocument()) { selectedDocument.set(it) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        RuntimeBridge.install(this)
        requestPermissions()
        setContent { App() }
    }

    private fun requestPermissions() {
        val requested = buildList {
            if (Build.VERSION.SDK_INT >= 31) {
                add(Manifest.permission.BLUETOOTH_CONNECT)
                add(Manifest.permission.BLUETOOTH_SCAN)
            }
            if (Build.VERSION.SDK_INT >= 33) add(Manifest.permission.POST_NOTIFICATIONS)
        }.filter { ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED }
        if (requested.isNotEmpty()) permissions.launch(requested.toTypedArray())
    }

    @Composable
    private fun App() {
        val runtime by RuntimeBridge.state.collectAsState()
        val loaded = remember { runCatching { SettingsStore(this).load() } }
        if (loaded.isFailure) {
            SettingsRecovery(loaded.exceptionOrNull()?.message ?: "settings cannot be loaded")
            return
        }
        val original = loaded.getOrThrow()
        var tab by remember { mutableStateOf("会话") }
        var mode by remember { mutableStateOf(original.mode) }
        var nodeId by remember { mutableStateOf(original.nodeId.toString()) }
        var parentId by remember { mutableStateOf(original.parentId.toString()) }
        var transport by remember { mutableStateOf(original.transport) }
        var endpoint by remember { mutableStateOf(original.endpoint) }
        var parentKey by remember { mutableStateOf(original.parentPublicKey) }
        var permit by remember { mutableStateOf(original.permitJson) }
        var tcpListen by remember { mutableStateOf(original.tcpListen) }
        var rfcommListen by remember { mutableStateOf(original.rfcommListen) }
        val settings = {
            SettingsV1(mode = mode, nodeId = nodeId.toLong(), parentId = parentId.toLong(), transport = transport,
                endpoint = endpoint.trim(), parentPublicKey = parentKey.trim(), permitJson = permit.trim(),
                tcpListen = tcpListen.trim(), rfcommListen = rfcommListen.trim())
        }
        val background = Color(0xFF08111F)
        MaterialTheme {
            Surface(Modifier.fillMaxSize(), color = background) {
                Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                    Text("MYFLOWHUB / CANONICAL ANDROID", color = Color(0xFF57D9B5), style = MaterialTheme.typography.labelSmall)
                    Text("节点树控制台", color = Color(0xFFEAF1FC), style = MaterialTheme.typography.headlineLarge)
                    Text("${runtime.mode} · ${runtime.connection}", color = if (runtime.error.isBlank()) Color(0xFF93A5BE) else Color(0xFFFF879A))
                    Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        listOf("会话", "资源", "文件", "状态").forEach { item ->
                            Button(onClick = { tab = item }, colors = ButtonDefaults.buttonColors(containerColor = if (tab == item) Color(0xFF287D70) else Color(0xFF1A2B43))) { Text(item) }
                        }
                    }
                    when (tab) {
                        "会话" -> SessionPage(mode, { mode = it }, nodeId, { nodeId = it }, parentId, { parentId = it }, transport, { transport = it }, endpoint, { endpoint = it }, parentKey, { parentKey = it }, permit, { permit = it }, tcpListen, { tcpListen = it }, rfcommListen, { rfcommListen = it }, settings)
                        "资源" -> ResourcesPage(runtime)
                        "文件" -> FilesPage(runtime)
                        else -> StatusPage(runtime)
                    }
                    if (runtime.error.isNotBlank()) JsonCard("错误（平台或 authority 明确返回）", runtime.error, Color(0xFF4B2330))
                }
            }
        }
    }

    @Composable
    private fun SettingsRecovery(message: String) {
        var confirmation by remember { mutableStateOf("") }
        var resetError by remember { mutableStateOf("") }
        Surface(Modifier.fillMaxSize(), color = Color(0xFF08111F)) {
            Column(Modifier.padding(24.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                Text("设置需要显式重置", color = Color(0xFFFF879A), style = MaterialTheme.typography.headlineMedium)
                Text(message, color = Color(0xFFB9C7D9))
                Text("身份、信任和权限状态不会被删除。请输入 RESET ANDROID V1。", color = Color(0xFF93A5BE))
                Input("确认文本", confirmation, { confirmation = it })
                Button(onClick = {
                    runCatching { SettingsStore(this@MainActivity).reset(confirmation) }
                        .onSuccess { recreate() }.onFailure { resetError = it.message ?: it.toString() }
                }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF8F4054))) { Text("重置连接设置") }
                if (resetError.isNotBlank()) Text(resetError, color = Color(0xFFFF879A))
            }
        }
    }

    @Composable
    private fun SessionPage(
        mode: String, setMode: (String) -> Unit, nodeId: String, setNodeId: (String) -> Unit,
        parentId: String, setParentId: (String) -> Unit, transport: String, setTransport: (String) -> Unit,
        endpoint: String, setEndpoint: (String) -> Unit, key: String, setKey: (String) -> Unit,
        permit: String, setPermit: (String) -> Unit, tcpListen: String, setTCPListen: (String) -> Unit,
        rfcommListen: String, setRFCOMMListen: (String) -> Unit, settings: () -> SettingsV1,
    ) {
        CardPanel("身份、链路与本地 Hub") {
            ToggleRow("模式", mode, "client", "host", setMode)
            ToggleRow("父链路", transport, "tcp", "rfcomm", setTransport)
            Input("本机 NodeID", nodeId, setNodeId)
            Input("父 NodeID", parentId, setParentId)
            Input("父端点", endpoint, setEndpoint)
            Input("父公钥", key, setKey)
            Input("首次准入 permit JSON", permit, setPermit, 3)
            if (mode == "host") {
                Input("TCP listener（留空禁用）", tcpListen, setTCPListen)
                Input("RFCOMM listener（留空禁用）", rfcommListen, setRFCOMMListen)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(9.dp)) {
                Button(onClick = { runIO {
                    val value = settings()
                    SettingsStore(this@MainActivity).save(value)
                    keepRuntime()
                    if (value.mode == "host") RuntimeBridge.startHost(this@MainActivity, value) else RuntimeBridge.startClient(this@MainActivity, value)
                } }) { Text("保存并启动") }
                Button(onClick = { stopRuntime() }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF623244))) { Text("停止") }
            }
        }
    }

    @Composable
    private fun ResourcesPage(runtime: RuntimeState) {
        var owner by remember { mutableStateOf("1") }
        var name by remember { mutableStateOf("system/health") }
        var request by remember { mutableStateOf("{\"version\":1}") }
        CardPanel("资源目录：Variable · Stream · Command") {
            Input("Owner NodeID", owner, { owner = it })
            Input("资源名", name, { name = it })
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { runIO { RuntimeBridge.catalog(owner.toLong()) } }) { Text("目录") }
                Button(onClick = { runIO { RuntimeBridge.snapshot(owner.toLong(), name) } }) { Text("快照") }
                Button(onClick = { runIO { RuntimeBridge.subscribe(owner.toLong(), name) } }) { Text("订阅") }
            }
            Input("Command JSON", request, { request = it }, 4)
            Button(onClick = { runIO { RuntimeBridge.invoke(owner.toLong(), name, request) } }) { Text("执行 Command") }
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                listOf("system/topology", "metrics/config", "clipboard/status", "file/transfers").forEach { resource ->
                    Button(onClick = { name = resource }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF1E3851))) { Text(resource) }
                }
            }
        }
        JsonCard("目录", runtime.catalogJson)
        JsonCard("结果", runtime.resultJson)
        if (runtime.lastEventJson.isNotBlank()) JsonCard("最新订阅事件", runtime.lastEventJson)
    }

    @Composable
    private fun FilesPage(runtime: RuntimeState) {
        var owner by remember { mutableStateOf("1") }
        var destination by remember { mutableStateOf("inbox/mobile-upload.bin") }
        var contentType by remember { mutableStateOf("application/octet-stream") }
        CardPanel("File feature") {
            Text("选择文档后复制到应用临时目录，由 canonical SDK 执行 64 KiB 分块、摘要和失败取消。", color = Color(0xFF93A5BE))
            Input("Owner NodeID", owner, { owner = it })
            Input("目标相对路径", destination, { destination = it })
            Input("Content-Type", contentType, { contentType = it })
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { documentPicker.launch(arrayOf("*/*")) }) { Text("选择文件") }
                Button(onClick = { runIO {
                    val uri = selectedDocument.get() ?: throw IllegalArgumentException("请先选择文件")
                    RuntimeBridge.upload(this@MainActivity, uri, owner.toLong(), destination, contentType)
                } }) { Text("上传") }
                Button(onClick = { runIO { RuntimeBridge.snapshot(owner.toLong(), "file/transfers") } }) { Text("传输列表") }
            }
        }
        JsonCard("文件结果", runtime.resultJson)
    }

    @Composable
    private fun StatusPage(runtime: RuntimeState) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = { runIO { RuntimeBridge.refreshStatus() } }) { Text("刷新") }
            Button(onClick = { requestPermissions() }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF1E3851))) { Text("请求蓝牙权限") }
        }
        JsonCard("身份", runtime.identityJson.ifBlank { "{}" })
        JsonCard("运行状态", runtime.statusJson)
    }

    @Composable
    private fun ToggleRow(label: String, current: String, first: String, second: String, update: (String) -> Unit) {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(label, color = Color(0xFF93A5BE), style = MaterialTheme.typography.labelMedium)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                listOf(first, second).forEach { value -> Button(onClick = { update(value) }, colors = ButtonDefaults.buttonColors(containerColor = if (current == value) Color(0xFF287D70) else Color(0xFF1A2B43))) { Text(value) } }
            }
        }
    }

    @Composable
    private fun Input(label: String, value: String, update: (String) -> Unit, lines: Int = 1) {
        OutlinedTextField(value = value, onValueChange = update, label = { Text(label) }, minLines = lines, modifier = Modifier.fillMaxWidth())
    }

    @Composable
    private fun CardPanel(title: String, content: @Composable () -> Unit) {
        Card(colors = CardDefaults.cardColors(containerColor = Color(0xFF101D30)), modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(15.dp), verticalArrangement = Arrangement.spacedBy(11.dp)) {
                Text(title, color = Color(0xFF62DDBB), style = MaterialTheme.typography.titleMedium)
                content()
            }
        }
    }

    @Composable
    private fun JsonCard(title: String, value: String, color: Color = Color(0xFF0D192A)) {
        Card(colors = CardDefaults.cardColors(containerColor = color), modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(15.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(title, color = Color(0xFF62DDBB))
                Text(value, color = Color(0xFFB9C7D9), style = MaterialTheme.typography.bodySmall)
            }
        }
    }

    private fun runIO(block: () -> Unit) {
        lifecycleScope.launch(Dispatchers.IO) { runCatching(block).onFailure(RuntimeBridge::reportError) }
    }

    private fun keepRuntime() {
        ContextCompat.startForegroundService(this, Intent(this, NodeService::class.java).setAction(NodeService.ACTION_KEEP))
    }

    private fun stopRuntime() {
        ContextCompat.startForegroundService(this, Intent(this, NodeService::class.java).setAction(NodeService.ACTION_STOP))
    }
}
