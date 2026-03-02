import SwiftUI

// Route polyline rendered on MapLibre layer, not as SwiftUI overlay.
// This file contains the style configuration and animation helpers.

struct RouteStyle {
    static let lineColor = Color.accentBlue
    static let lineWidth: CGFloat = 4.0
    static let glowColor = Color.accentBlue.opacity(0.3)
    static let glowWidth: CGFloat = 8.0
    static let animationDuration: TimeInterval = 0.8
}
