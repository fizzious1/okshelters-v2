import SwiftUI

struct StatusBadge: View {
    let status: ShelterStatus

    var body: some View {
        Circle()
            .fill(status.color)
            .frame(width: 10, height: 10)
    }
}

struct CapacityBar: View {
    let occupancy: Int
    let capacity: Int

    private var fillFraction: Double {
        guard capacity > 0 else { return 0 }
        return Double(occupancy) / Double(capacity)
    }

    private var fillColor: Color {
        switch fillFraction {
        case ..<0.7: return .accentGreen
        case ..<0.9: return .accentAmber
        default: return .accentRed
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(Color.bgSurfaceRaised)
                        .frame(height: 6)

                    RoundedRectangle(cornerRadius: 3)
                        .fill(fillColor)
                        .frame(
                            width: geometry.size.width * fillFraction,
                            height: 6
                        )
                }
            }
            .frame(height: 6)

            Text("\(Int(fillFraction * 100))% full")
                .font(.caption2)
                .foregroundStyle(Color.textSecondary)
        }
    }
}
