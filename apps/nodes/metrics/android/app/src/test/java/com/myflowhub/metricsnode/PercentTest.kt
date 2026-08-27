package com.myflowhub.metricsnode

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class PercentTest {
    @Test fun scalesAndClampsWithinDeviceRange() {
        assertEquals(128, Percent.value("50", 255))
        assertEquals(0, Percent.value("0", 15))
        assertEquals(15, Percent.value("100", 15))
    }

    @Test fun rejectsOutOfRangeValues() {
        assertThrows(IllegalArgumentException::class.java) { Percent.value("101", 255) }
    }
}
