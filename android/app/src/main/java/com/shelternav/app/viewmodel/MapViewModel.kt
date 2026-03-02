package com.shelternav.app.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.shelternav.app.LatLon
import com.shelternav.app.ShelterInfo
import com.shelternav.app.bridge.ShelterNavJNI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class MapViewModel : ViewModel() {
    private val _nearbyShelters = MutableStateFlow<List<ShelterInfo>>(emptyList())
    val nearbyShelters: StateFlow<List<ShelterInfo>> = _nearbyShelters.asStateFlow()

    private val _selectedShelter = MutableStateFlow<ShelterInfo?>(null)
    val selectedShelter: StateFlow<ShelterInfo?> = _selectedShelter.asStateFlow()

    private val _userLocation = MutableStateFlow<LatLon?>(null)
    val userLocation: StateFlow<LatLon?> = _userLocation.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    fun search(query: String) {
        // TODO: Filter shelters by name/address
    }

    fun findNearest() {
        val location = _userLocation.value ?: return
        viewModelScope.launch(Dispatchers.Default) {
            _isLoading.value = true
            _nearbyShelters.value = ShelterNavJNI.findNearest(
                lat = location.lat,
                lon = location.lon,
                radiusM = 5000,
                limit = 10,
            )
            _isLoading.value = false
        }
    }

    fun selectShelter(shelter: ShelterInfo) {
        _selectedShelter.value = shelter
    }

    fun updateUserLocation(lat: Double, lon: Double) {
        _userLocation.value = LatLon(lat, lon)
    }
}
