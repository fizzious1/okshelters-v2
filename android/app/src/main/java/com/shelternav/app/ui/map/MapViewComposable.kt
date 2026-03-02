package com.shelternav.app.ui.map

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import com.shelternav.app.viewmodel.MapViewModel

// MapLibre Native Android wrapper
@Composable
fun MapViewComposable(
    modifier: Modifier = Modifier,
    mapViewModel: MapViewModel,
) {
    AndroidView(
        modifier = modifier,
        factory = { context ->
            // TODO: Initialize MapLibre MapView
            // val mapView = MapView(context)
            // mapView.getMapAsync { map ->
            //     map.setStyle(shelterNavDarkStyle)
            //     map.locationComponent.activateLocationComponent(...)
            // }
            android.view.View(context)
        },
        update = { _ ->
            // TODO: Update map annotations from viewModel state
        }
    )
}
