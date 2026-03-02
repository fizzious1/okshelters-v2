import SwiftUI
import CoreLocation

struct ManeuverInfo: Identifiable {
    let id = UUID()
    let instruction: String
    let distanceM: Double

    var distanceText: String {
        if distanceM < 1000 {
            return "in \(Int(distanceM))m"
        }
        return String(format: "in %.1fkm", distanceM / 1000)
    }
}

@MainActor
final class NavigationViewModel: ObservableObject {
    @Published var isNavigating = false
    @Published var destination: ShelterInfo?
    @Published var nextManeuver: ManeuverInfo?
    @Published var estimatedArrival: String?
    @Published var routeCoordinates: [CLLocationCoordinate2D] = []

    private let bridge: ShelterNavBridge

    init(bridge: ShelterNavBridge = .shared) {
        self.bridge = bridge
    }

    func startNavigation(to shelter: ShelterInfo, from location: CLLocationCoordinate2D) {
        destination = shelter
        isNavigating = true
        Haptics.navigationStarted()

        Task {
            let route = await bridge.getRoute(
                startLat: location.latitude,
                startLon: location.longitude,
                endLat: shelter.lat,
                endLon: shelter.lon
            )
            routeCoordinates = route.path
            estimatedArrival = "\(route.estimatedSeconds / 60) min"
            nextManeuver = ManeuverInfo(
                instruction: "Continue to \(shelter.name)",
                distanceM: route.totalDistanceM
            )
        }
    }

    func stopNavigation() {
        isNavigating = false
        destination = nil
        nextManeuver = nil
        estimatedArrival = nil
        routeCoordinates = []
    }
}
