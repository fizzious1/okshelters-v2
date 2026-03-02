package com.shelternav.app.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.shelternav.app.LatLon
import com.shelternav.app.ManeuverInfo
import com.shelternav.app.ShelterInfo
import com.shelternav.app.bridge.ShelterNavJNI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class NavigationViewModel : ViewModel() {
    private val _isNavigating = MutableStateFlow(false)
    val isNavigating: StateFlow<Boolean> = _isNavigating.asStateFlow()

    private val _destination = MutableStateFlow<ShelterInfo?>(null)
    val destination: StateFlow<ShelterInfo?> = _destination.asStateFlow()

    private val _nextManeuver = MutableStateFlow<ManeuverInfo?>(null)
    val nextManeuver: StateFlow<ManeuverInfo?> = _nextManeuver.asStateFlow()

    private val _estimatedArrival = MutableStateFlow<String?>(null)
    val estimatedArrival: StateFlow<String?> = _estimatedArrival.asStateFlow()

    private val _routeCoordinates = MutableStateFlow<List<LatLon>>(emptyList())
    val routeCoordinates: StateFlow<List<LatLon>> = _routeCoordinates.asStateFlow()

    fun startNavigation(shelter: ShelterInfo, from: LatLon) {
        _destination.value = shelter
        _isNavigating.value = true

        viewModelScope.launch(Dispatchers.Default) {
            val route = ShelterNavJNI.getRoute(
                startLat = from.lat,
                startLon = from.lon,
                endLat = shelter.lat,
                endLon = shelter.lon,
            )
            _routeCoordinates.value = route.path
            _estimatedArrival.value = "${route.estimatedSeconds / 60} min"
        }
    }

    fun stopNavigation() {
        _isNavigating.value = false
        _destination.value = null
        _nextManeuver.value = null
        _estimatedArrival.value = null
        _routeCoordinates.value = emptyList()
    }
}
