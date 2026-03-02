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

    private let bridge = ShelterNavBridge()

    func startNavigation(to shelter: ShelterInfo, from location: CLLocationCoordinate2D) {
        destination = shelter
        isNavigating = true

        Task {
            let route = await bridge.getRoute(
                startLat: location.latitude,
                startLon: location.longitude,
                endLat: shelter.lat,
                endLon: shelter.lon
            )
            routeCoordinates = route.path
            estimatedArrival = "\(route.estimatedSeconds / 60) min"
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
