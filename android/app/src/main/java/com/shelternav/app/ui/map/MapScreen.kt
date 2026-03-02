package com.shelternav.app.ui.map

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.shelternav.app.ui.nav.NavigationOverlay
import com.shelternav.app.ui.search.SearchBar
import com.shelternav.app.ui.sheet.ShelterDetailSheet
import com.shelternav.app.viewmodel.MapViewModel
import com.shelternav.app.viewmodel.NavigationViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MapScreen(
    mapViewModel: MapViewModel,
    navigationViewModel: NavigationViewModel,
) {
    val isNavigating by navigationViewModel.isNavigating.collectAsState()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = false)

    Scaffold { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            // Full-screen map
            MapViewComposable(
                modifier = Modifier.fillMaxSize(),
                mapViewModel = mapViewModel,
            )

            // Search bar overlay
            if (!isNavigating) {
                SearchBar(
                    modifier = Modifier
                        .align(Alignment.TopCenter)
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                    onSearch = { mapViewModel.search(it) },
                )
            }

            // Navigation overlay
            if (isNavigating) {
                NavigationOverlay(
                    navigationViewModel = navigationViewModel,
                )
            }
        }

        // Bottom sheet
        ModalBottomSheet(
            sheetState = sheetState,
            onDismissRequest = {},
        ) {
            ShelterDetailSheet(mapViewModel = mapViewModel)
        }
    }
}
