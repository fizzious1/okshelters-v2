import SwiftUI

struct ShelterSheet: View {
    @EnvironmentObject var mapViewModel: MapViewModel

    var body: some View {
        NavigationStack {
            List {
                if mapViewModel.nearbyShelters.isEmpty {
                    ContentUnavailableView(
                        "No Shelters Found",
                        systemImage: "mappin.slash",
                        description: Text("Move the map or increase search radius")
                    )
                } else {
                    ForEach(mapViewModel.nearbyShelters) { shelter in
                        ShelterCard(shelter: shelter)
                    }
                }
            }
            .listStyle(.plain)
            .navigationTitle("Nearby Shelters")
            .navigationBarTitleDisplayMode(.inline)
            .task {
                mapViewModel.findNearestIfNeeded()
            }
            .refreshable {
                await mapViewModel.refreshData()
            }
        }
    }
}

struct ShelterCard: View {
    @EnvironmentObject private var mapViewModel: MapViewModel
    @EnvironmentObject private var navigationViewModel: NavigationViewModel
    @Environment(\.openURL) private var openURL

    let shelter: ShelterInfo

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                StatusBadge(status: shelter.status)
                Text(shelter.name)
                    .font(.headline)
                    .foregroundStyle(Color.textPrimary)
                Spacer()
                Text(shelter.formattedDistance)
                    .font(.subheadline)
                    .foregroundStyle(Color.textSecondary)
            }

            Text(shelter.address)
                .font(.subheadline)
                .foregroundStyle(Color.textSecondary)

            CapacityBar(
                occupancy: shelter.occupancy,
                capacity: shelter.capacity
            )

            HStack(spacing: 12) {
                Button("Navigate") {
                    guard let location = mapViewModel.userLocation else { return }
                    navigationViewModel.startNavigation(to: shelter, from: location)
                }
                .buttonStyle(.borderedProminent)
                .tint(Color.accentBlue)
                .disabled(mapViewModel.userLocation == nil)

                if let phone = shelter.phone {
                    Button("Call") {
                        guard let telURL = makeTelURL(from: phone) else { return }
                        openURL(telURL)
                    }
                        .buttonStyle(.bordered)
                }
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onTapGesture {
            mapViewModel.selectShelter(shelter)
        }
    }

    private func makeTelURL(from phone: String) -> URL? {
        let allowed = phone.filter { $0.isNumber || $0 == "+" }
        return URL(string: "tel://\(allowed)")
    }
}
