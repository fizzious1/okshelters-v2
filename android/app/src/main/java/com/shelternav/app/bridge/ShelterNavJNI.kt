package com.shelternav.app.bridge

import com.shelternav.app.LatLon
import com.shelternav.app.ShelterInfo
import com.shelternav.app.ShelterStatus
import com.shelternav.app.ShelterType

data class RouteResult(
    val path: List<LatLon>,
    val totalDistanceM: Double,
    val estimatedSeconds: Int,
)

// JNI bridge to libshelternav C library
object ShelterNavJNI {

    init {
        System.loadLibrary("shelternav")
    }

    // Native methods — implemented in libshelternav via JNI
    private external fun nativeInit(dbPath: String): Int
    private external fun nativeClose()
    private external fun nativeFindNearest(
        lat: Double, lon: Double, radiusM: Double, limit: Int
    ): Array<IntArray> // Simplified; real impl returns structured data

    private external fun nativeGetRoute(
        startLat: Double, startLon: Double, endLat: Double, endLon: Double
    ): DoubleArray

    fun init(dbPath: String) {
        val result = nativeInit(dbPath)
        check(result == 0) { "Failed to initialize libshelternav: error $result" }
    }

    fun close() {
        nativeClose()
    }

    fun findNearest(lat: Double, lon: Double, radiusM: Int, limit: Int): List<ShelterInfo> {
        // TODO: Call native and map results
        return emptyList()
    }

    fun getRoute(startLat: Double, startLon: Double, endLat: Double, endLon: Double): RouteResult {
        // TODO: Call native and map results
        return RouteResult(path = emptyList(), totalDistanceM = 0.0, estimatedSeconds = 0)
    }
}
