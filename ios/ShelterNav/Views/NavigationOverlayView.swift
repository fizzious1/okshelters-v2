import SwiftUI

struct NavigationOverlayView: View {
    @EnvironmentObject var navigationViewModel: NavigationViewModel

    var body: some View {
        VStack {
            // Top bar
            HStack {
                Button(action: { navigationViewModel.stopNavigation() }) {
                    Image(systemName: "chevron.left")
                    Text("Back")
                }
                .foregroundStyle(Color.textPrimary)

                Spacer()

                Text("NAVIGATING")
                    .font(.caption)
                    .fontWeight(.bold)
                    .foregroundStyle(Color.accentBlue)

                Spacer()

                if let eta = navigationViewModel.estimatedArrival {
                    Text("ETA \(eta)")
                        .font(.subheadline)
                        .foregroundStyle(Color.textSecondary)
                }
            }
            .padding()
            .background(.ultraThinMaterial)

            Spacer()

            // Next maneuver card
            if let maneuver = navigationViewModel.nextManeuver {
                VStack(alignment: .leading, spacing: 4) {
                    Text(maneuver.instruction)
                        .font(.title3)
                        .fontWeight(.semibold)
                        .foregroundStyle(Color.textPrimary)
                    Text(maneuver.distanceText)
                        .font(.subheadline)
                        .foregroundStyle(Color.textSecondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding()
                .background(Color.bgSurface)
                .clipShape(RoundedRectangle(cornerRadius: 16))
                .padding(.horizontal)
            }

            // Destination card
            if let destination = navigationViewModel.destination {
                HStack {
                    StatusBadge(status: destination.status)
                    VStack(alignment: .leading) {
                        Text(destination.name)
                            .font(.headline)
                            .foregroundStyle(Color.textPrimary)
                        Text(destination.formattedDistance)
                            .font(.subheadline)
                            .foregroundStyle(Color.textSecondary)
                    }
                    Spacer()
                }
                .padding()
                .background(Color.bgSurface)
                .clipShape(RoundedRectangle(cornerRadius: 16))
                .padding(.horizontal)
                .padding(.bottom, 16)
            }
        }
    }
}
