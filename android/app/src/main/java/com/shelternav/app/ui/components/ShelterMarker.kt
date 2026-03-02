package com.shelternav.app.ui.components

import com.shelternav.app.ShelterStatus

// Shelter markers are rendered as MapLibre symbol layers, not Compose views.
// This file contains marker style configuration.

object ShelterMarkerStyle {
    fun iconColor(status: ShelterStatus): String = when (status) {
        ShelterStatus.OPEN -> "#34D399"
        ShelterStatus.FULL -> "#FBBF24"
        ShelterStatus.CLOSED -> "#F87171"
    }

    const val ICON_SIZE = 0.8f
    const val CLUSTER_RADIUS = 50
    const val CLUSTER_MAX_ZOOM = 13
}
