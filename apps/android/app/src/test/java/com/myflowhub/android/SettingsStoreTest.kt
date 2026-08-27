package com.myflowhub.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class SettingsStoreTest {
    @Test fun settingsRoundTripIsVersioned() {
        val value = SettingsV1(nodeId = 44, transport = "rfcomm", endpoint = "bt+rfcomm://AA:BB:CC:DD:EE:FF")
        assertEquals(value, SettingsStore.decode(SettingsStore.encode(value)))
    }

    @Test fun settingsRejectUnknownVersionAndInvalidIds() {
        assertThrows(IllegalArgumentException::class.java) { SettingsStore.decode("{\"version\":2}") }
        assertThrows(IllegalArgumentException::class.java) { SettingsStore.encode(SettingsV1(nodeId = 0)) }
    }
}
