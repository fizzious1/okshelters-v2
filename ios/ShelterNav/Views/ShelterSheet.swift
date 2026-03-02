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
        }
    }
}

struct ShelterCard: View {
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
                Button("Navigate") {}
                    .buttonStyle(.borderedProminent)
                    .tint(Color.accentBlue)

                if let phone = shelter.phone {
                    Button("Call") {}
                        .buttonStyle(.bordered)
                }
            }
        }
        .padding(.vertical, 4)
    }
}
