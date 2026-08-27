package com.myflowhub.metricsnode

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
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
import org.json.JSONObject
import java.io.File

class MainActivity : ComponentActivity() {
    private val permissions = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestRuntimePermissions()
        setContent { MetricsScreen() }
    }

    private fun requestRuntimePermissions() {
        val requested = buildList {
            add(Manifest.permission.CAMERA)
            if (Build.VERSION.SDK_INT >= 33) add(Manifest.permission.POST_NOTIFICATIONS)
        }.filter { ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED }
        if (requested.isNotEmpty()) permissions.launch(requested.toTypedArray())
    }

    @Composable
    private fun MetricsScreen() {
        val state by NodeStateStore.state.collectAsState()
        var nodeID by remember { mutableStateOf("20") }
        var parentID by remember { mutableStateOf("1") }
        var endpoint by remember { mutableStateOf("10.0.2.2:7341") }
        var parentKey by remember { mutableStateOf("") }
        var permit by remember { mutableStateOf("") }
        val background = Color(0xFF0C121E)
        MaterialTheme {
            Surface(modifier = Modifier.fillMaxSize(), color = background) {
                Column(
                    modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(22.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    Text("MYFLOWHUB / NODE", color = Color(0xFF5DC9C4), style = MaterialTheme.typography.labelSmall)
                    Text("Metrics", color = Color(0xFFE8EDF7), style = MaterialTheme.typography.headlineLarge)
                    Text("连接状态：${state.connection}", color = Color(0xFF9BADC6))
                    Input("节点 ID", nodeID) { nodeID = it }
                    Input("父节点 ID", parentID) { parentID = it }
                    Input("TCP 端点", endpoint) { endpoint = it }
                    Input("父节点公钥", parentKey) { parentKey = it }
                    Input("一次性接入凭证（可选 JSON）", permit, 3) { permit = it }
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        Button(onClick = { readIdentity(nodeID) }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF28425A))) { Text("读取身份") }
                        Button(onClick = { startNode(nodeID, parentID, endpoint, parentKey, permit) }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF39AFA6))) { Text("启动") }
                        Button(onClick = { stopNode() }, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF553342))) { Text("停止") }
                    }
                    Button(onClick = { startActivity(Intent(Settings.ACTION_MANAGE_WRITE_SETTINGS, Uri.parse("package:$packageName"))) }) { Text("授予亮度控制权限") }
                    if (state.error.isNotBlank()) Text(state.error, color = Color(0xFFFF8B8B))
                    if (state.identityJSON.isNotBlank()) JsonCard("身份", state.identityJSON)
                    if (state.statusJSON.isNotBlank()) JsonCard("资源快照", state.statusJSON)
                    if (!bindingAvailable()) Text("metricsmobile.aar 未安装：Gradle 可验证 UI，但运行节点前必须生成 canonical AAR。", color = Color(0xFFEDB457))
                }
            }
        }
    }

    @Composable
    private fun Input(label: String, value: String, lines: Int = 1, update: (String) -> Unit) {
        OutlinedTextField(value = value, onValueChange = update, label = { Text(label) }, minLines = lines, modifier = Modifier.fillMaxWidth())
    }

    @Composable
    private fun JsonCard(title: String, value: String) {
        Card(colors = CardDefaults.cardColors(containerColor = Color(0xFF111B2A)), modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(title, color = Color(0xFF5DC9C4))
                Text(pretty(value), color = Color(0xFFB8C5D8), style = MaterialTheme.typography.bodySmall)
            }
        }
    }

    private fun readIdentity(nodeID: String) {
        Thread {
            runCatching {
                MobileBridge().identity(JSONObject().put("version", 1).put("state_directory", File(filesDir, "metrics-vnext").absolutePath).put("node_id", nodeID).toString())
            }.onSuccess(NodeStateStore::identity).onFailure(NodeStateStore::error)
        }.start()
    }

    private fun startNode(nodeID: String, parentID: String, endpoint: String, parentKey: String, permit: String) {
        val intent = Intent(this, NodeService::class.java).setAction(NodeService.ACTION_START)
            .putExtra(NodeService.EXTRA_NODE_ID, nodeID).putExtra(NodeService.EXTRA_PARENT_ID, parentID)
            .putExtra(NodeService.EXTRA_ENDPOINT, endpoint).putExtra(NodeService.EXTRA_PARENT_KEY, parentKey)
            .putExtra(NodeService.EXTRA_PERMIT, permit)
        ContextCompat.startForegroundService(this, intent)
    }

    private fun stopNode() {
        ContextCompat.startForegroundService(this, Intent(this, NodeService::class.java).setAction(NodeService.ACTION_STOP))
    }

    private fun bindingAvailable(): Boolean = runCatching { Class.forName("metricsmobile.Metricsmobile") }.isSuccess
    private fun pretty(raw: String): String = runCatching { JSONObject(raw).toString(2) }.getOrDefault(raw)
}
