package com.myflowhub.metricsnode

/** Serializes service runtime replacement without persisting live sessions. */
internal class RuntimeGeneration {
    private var value = 0L

    @Synchronized
    fun begin(replace: () -> Unit): Long {
        value += 1
        replace()
        return value
    }

    @Synchronized
    fun invalidate(clear: () -> Unit) {
        value += 1
        clear()
    }

    @Synchronized
    fun commit(candidate: Long, install: () -> Unit): Boolean {
        if (candidate != value) return false
        install()
        return true
    }

    @Synchronized
    fun isCurrent(candidate: Long): Boolean = candidate == value
}
