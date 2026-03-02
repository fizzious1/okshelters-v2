import SwiftUI

struct MapContainerView: View {
    @EnvironmentObject var mapViewModel: MapViewModel
    @EnvironmentObject var navigationViewModel: NavigationViewModel

    var body: some View {
        ZStack {
            MapView()
                .ignoresSafeArea()

            VStack {
                SearchBar()
                    .padding(.horizontal)
                    .padding(.top, 8)

                Spacer()

                if navigationViewModel.isNavigating {
                    NavigationOverlayView()
                } else {
                    HStack {
                        Spacer()
                        VStack(spacing: 12) {
                            RecenterButton(action: mapViewModel.recenterOnUser)
                            NearestButton(action: mapViewModel.findNearest)
                        }
                        .padding(.trailing, 16)
                        .padding(.bottom, 140)
                    }
                }
            }
        }
        .sheet(isPresented: .constant(true)) {
            ShelterSheet()
                .presentationDetents([.height(120), .medium, .large])
                .presentationDragIndicator(.visible)
                .presentationBackgroundInteraction(.enabled)
                .interactiveDismissDisabled()
        }
    }
}

private struct RecenterButton: View {
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: "location.fill")
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(Color.accentBlue)
                .frame(width: 44, height: 44)
                .background(.ultraThinMaterial)
                .clipShape(Circle())
        }
    }
}

private struct NearestButton: View {
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: "bolt.fill")
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(Color.accentGreen)
                .frame(width: 44, height: 44)
                .background(.ultraThinMaterial)
                .clipShape(Circle())
        }
    }
}
