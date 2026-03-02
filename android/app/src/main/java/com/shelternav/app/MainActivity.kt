package com.shelternav.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import com.shelternav.app.theme.ShelterNavTheme
import com.shelternav.app.ui.map.MapScreen
import com.shelternav.app.viewmodel.MapViewModel
import com.shelternav.app.viewmodel.NavigationViewModel

class MainActivity : ComponentActivity() {
    private val mapViewModel: MapViewModel by viewModels()
    private val navigationViewModel: NavigationViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            ShelterNavTheme {
                MapScreen(
                    mapViewModel = mapViewModel,
                    navigationViewModel = navigationViewModel,
                )
            }
        }
    }
}
