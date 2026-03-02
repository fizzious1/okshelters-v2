import SwiftUI

// Uses SF Pro (system font) — no custom font loading needed
extension Font {
    static let shelterTitle = Font.system(.headline, weight: .semibold)
    static let shelterSubtitle = Font.system(.subheadline)
    static let shelterCaption = Font.system(.caption2)
    static let navInstruction = Font.system(.title3, weight: .semibold)
    static let navEta = Font.system(.subheadline)
}
