import SwiftUI

enum ShelterStatus: Int {
    case closed = 0
    case open = 1
    case full = 2

    var color: Color {
        switch self {
        case .closed: return .accentRed
        case .open: return .accentGreen
        case .full: return .accentAmber
        }
    }

    var label: String {
        switch self {
        case .closed: return "Closed"
        case .open: return "Open"
        case .full: return "Full"
        }
    }
}

enum ShelterType: Int {
    case emergency = 0
    case overnight = 1
    case longTerm = 2
}

struct ShelterInfo: Identifiable {
    let id: Int32
    let name: String
    let lat: Double
    let lon: Double
    let type: ShelterType
    let capacity: Int
    let occupancy: Int
    let status: ShelterStatus
    let address: String
    let phone: String?
    let distanceM: Double

    var formattedDistance: String {
        if distanceM < 1000 {
            return "\(Int(distanceM))m"
        }
        return String(format: "%.1fkm", distanceM / 1000)
    }
}
