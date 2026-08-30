package com.myflowhub.metricsnode

import com.myflowhub.metrics.metricsmobile.Client
import com.myflowhub.metrics.metricsmobile.Metricsmobile

/** Typed Kotlin facade over the generated gomobile ABI. */
internal class MobileBridge {
    private val client: Client = requireNotNull(Metricsmobile.newClient()) {
        "metricsmobile.newClient returned null"
    }

    fun identity(request: String): String = client.identity(request)
    fun start(request: String): String = client.start(request)
    fun stop() = client.stop()
    fun status(): String = client.status()
    fun configuration(): String = client.configuration()
    fun updateConfiguration(request: String): String = client.updateConfiguration(request)
    fun updateMetric(metric: String, value: String, error: String) = client.updateMetric(metric, value, error)
    fun nextAction(): String = client.nextAction()
    fun nextNotification(): String = client.nextNotification()
    fun completeAction(actionID: String, actualValue: String, error: String) =
        client.completeAction(actionID, actualValue, error)
}
