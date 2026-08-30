package com.myflowhub.metricsnode

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class RuntimeGenerationTest {
    @Test
    fun staleLaunchCannotInstallAfterStopOrReplacement() {
        val gate = RuntimeGeneration()
        var installed = ""
        val first = gate.begin { installed = "replacing" }
        val second = gate.begin { installed = "replacing" }

        assertFalse(gate.commit(first) { installed = "first" })
        assertTrue(gate.commit(second) { installed = "second" })
        gate.invalidate { installed = "stopped" }
        assertFalse(gate.commit(second) { installed = "stale" })
        assertTrue(installed == "stopped")
    }
}
