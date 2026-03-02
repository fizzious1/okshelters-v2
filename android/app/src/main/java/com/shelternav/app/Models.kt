package com.shelternav.app

import androidx.compose.ui.graphics.Color
import com.shelternav.app.theme.AccentAmber
import com.shelternav.app.theme.AccentGreen
import com.shelternav.app.theme.AccentRed

enum class ShelterStatus(val value: Int) {
    CLOSED(0),
    OPEN(1),
    FULL(2);

    val color: Color
        get() = when (this) {
            CLOSED -> AccentRed
            OPEN -> AccentGreen
            FULL -> AccentAmber
        }

    val label: String
        get() = when (this) {
            CLOSED -> "Closed"
            OPEN -> "Open"
            FULL -> "Full"
        }

    companion object {
        fun fromInt(value: Int): ShelterStatus = entries.first { it.value == value }
    }
}

enum class ShelterType(val value: Int) {
    EMERGENCY(0),
    OVERNIGHT(1),
    LONG_TERM(2);
}

data class ShelterInfo(
    val id: Int,
    val name: String,
    val lat: Double,
    val lon: Double,
    val type: ShelterType,
    val capacity: Int,
    val occupancy: Int,
    val status: ShelterStatus,
    val address: String,
    val phone: String?,
    val distanceM: Double,
) {
    val formattedDistance: String
        get() = if (distanceM < 1000) {
            "${distanceM.toInt()}m"
        } else {
            "%.1fkm".format(distanceM / 1000)
        }
}

data class LatLon(val lat: Double, val lon: Double)

data class ManeuverInfo(
    val point: LatLon,
    val instruction: String,
    val distanceM: Double,
) {
    val distanceText: String
        get() = if (distanceM < 1000) "in ${distanceM.toInt()}m" else "in ${"%.1f".format(distanceM / 1000)}km"
}
