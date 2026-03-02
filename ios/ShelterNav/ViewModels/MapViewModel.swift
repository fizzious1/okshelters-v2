import SwiftUI
import CoreLocation

@MainActor
final class MapViewModel: ObservableObject {
    @Published var nearbyShelters: [ShelterInfo] = []
    @Published var selectedShelter: ShelterInfo?
    @Published var userLocation: CLLocationCoordinate2D?
    @Published var isLoading = false

    private let bridge = ShelterNavBridge()

    func search(query: String) {
        // TODO: Filter shelters by name/address
    }

    func findNearest() {
        guard let location = userLocation else { return }
        Task {
            isLoading = true
            defer { isLoading = false }
            nearbyShelters = await bridge.findNearest(
                lat: location.latitude,
                lon: location.longitude,
                radiusM: 5000,
                limit: 10
            )
        }
    }

    func selectShelter(_ shelter: ShelterInfo) {
        selectedShelter = shelter
    }
}
