package com.myflowhub.metricsnode

import java.lang.reflect.InvocationTargetException

internal class MobileBridge {
    private val client: Any
    private val type: Class<*>

    init {
        val entry = Class.forName("metricsmobile.Metricsmobile")
        client = requireNotNull(entry.getMethod("newClient").invoke(null)) { "metricsmobile.newClient returned null" }
        type = client.javaClass
    }

    fun identity(request: String): String = call("identity", request) as String
    fun start(request: String): String = call("start", request) as String
    fun stop() { call("stop") }
    fun status(): String = call("status") as String
    fun configuration(): String = call("configuration") as String
    fun updateConfiguration(request: String): String = call("updateConfiguration", request) as String
    fun updateMetric(metric: String, value: String, error: String) { call("updateMetric", metric, value, error) }
    fun nextAction(): String = call("nextAction") as String
    fun nextNotification(): String = call("nextNotification") as String
    fun completeAction(actionID: String, actualValue: String, error: String) {
        call("completeAction", actionID, actualValue, error)
    }

    private fun call(name: String, vararg arguments: String): Any? {
        try {
            val types = Array(arguments.size) { String::class.java }
            return type.getMethod(name, *types).invoke(client, *arguments)
        } catch (error: InvocationTargetException) {
            throw (error.targetException ?: error)
        }
    }
}
