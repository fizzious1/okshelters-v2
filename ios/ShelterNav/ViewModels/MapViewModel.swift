import SwiftUI
@preconcurrency import CoreLocation

@MainActor
final class MapViewModel: NSObject, ObservableObject {
    @Published var nearbyShelters: [ShelterInfo] = []
    @Published var selectedShelter: ShelterInfo?
    @Published var userLocation: CLLocationCoordinate2D?
    @Published var isLoading = false
    @Published private(set) var recenterRequestID = UUID()

    private var allShelters: [ShelterInfo] = []
    private var activeQuery = ""
    private var hasLoadedInitialShelters = false

    private let bridge: ShelterNavBridge
    private let locationManager = CLLocationManager()

    init(bridge: ShelterNavBridge = .shared) {
        self.bridge = bridge
        super.init()

        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyNearestTenMeters
        locationManager.distanceFilter = 25

        if locationManager.authorizationStatus == .notDetermined {
            locationManager.requestWhenInUseAuthorization()
        } else {
            locationManager.startUpdatingLocation()
        }

        Task {
            await bridge.syncDatabase()
            if userLocation != nil {
                findNearest()
            }
        }
    }

    func search(query: String) {
        activeQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
        applySearchFilter()
    }

    func findNearest() {
        guard let location = userLocation else { return }
        Task {
            await loadNearest(for: location)
        }
    }

    func findNearestIfNeeded() {
        guard !hasLoadedInitialShelters, !isLoading else { return }
        findNearest()
    }

    func refreshData() async {
        guard let userLocation else { return }
        await bridge.syncDatabase()
        await loadNearest(for: userLocation)
    }

    func recenterOnUser() {
        guard userLocation != nil else { return }
        Haptics.itemSelected()
        recenterRequestID = UUID()
    }

    func selectShelter(_ shelter: ShelterInfo) {
        selectedShelter = shelter
        Haptics.pinSelected()
    }

    private func applySearchFilter() {
        guard !activeQuery.isEmpty else {
            nearbyShelters = allShelters
            return
        }

        nearbyShelters = allShelters.filter { shelter in
            shelter.name.localizedCaseInsensitiveContains(activeQuery) ||
                shelter.address.localizedCaseInsensitiveContains(activeQuery)
        }
    }

    private func loadNearest(for location: CLLocationCoordinate2D) async {
        isLoading = true
        defer { isLoading = false }

        let shelters = await bridge.findNearest(
            lat: location.latitude,
            lon: location.longitude,
            radiusM: 5000,
            limit: 10
        )
        allShelters = shelters
        applySearchFilter()
        hasLoadedInitialShelters = true
    }
}

extension MapViewModel: CLLocationManagerDelegate {
    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            manager.startUpdatingLocation()
        case .denied, .restricted:
            manager.stopUpdatingLocation()
        case .notDetermined:
            manager.requestWhenInUseAuthorization()
        @unknown default:
            manager.stopUpdatingLocation()
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let coordinate = locations.last?.coordinate else { return }
        userLocation = coordinate
        findNearestIfNeeded()
    }
}
