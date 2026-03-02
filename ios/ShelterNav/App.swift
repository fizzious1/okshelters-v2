import SwiftUI

@main
struct ShelterNavApp: App {
    @StateObject private var mapViewModel = MapViewModel()
    @StateObject private var navigationViewModel = NavigationViewModel()

    init() {
        _ = ShelterNavBridge.shared
    }

    var body: some Scene {
        WindowGroup {
            MapContainerView()
                .environmentObject(mapViewModel)
                .environmentObject(navigationViewModel)
        }
    }
}
