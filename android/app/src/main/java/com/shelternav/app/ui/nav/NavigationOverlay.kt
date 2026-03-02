package com.shelternav.app.ui.nav

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.shelternav.app.ui.components.StatusBadge
import com.shelternav.app.viewmodel.NavigationViewModel

@Composable
fun NavigationOverlay(navigationViewModel: NavigationViewModel) {
    val destination by navigationViewModel.destination.collectAsState()
    val nextManeuver by navigationViewModel.nextManeuver.collectAsState()
    val eta by navigationViewModel.estimatedArrival.collectAsState()

    Column(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        // Top bar
        Surface(tonalElevation = 3.dp) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = { navigationViewModel.stopNavigation() }) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                }
                Spacer(modifier = Modifier.weight(1f))
                Text(
                    "NAVIGATING",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
                Spacer(modifier = Modifier.weight(1f))
                eta?.let {
                    Text(
                        "ETA $it",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }

        Column(modifier = Modifier.padding(16.dp)) {
            // Next maneuver
            nextManeuver?.let { maneuver ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            maneuver.instruction,
                            style = MaterialTheme.typography.titleMedium,
                        )
                        Text(
                            maneuver.distanceText,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }

            // Destination card
            destination?.let { dest ->
                Card(modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
                    Row(
                        modifier = Modifier.padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        StatusBadge(status = dest.status)
                        Column(modifier = Modifier.padding(start = 8.dp)) {
                            Text(dest.name, style = MaterialTheme.typography.titleSmall)
                            Text(
                                dest.formattedDistance,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
    }
}
