import UIKit

enum Haptics {
    private static let pinTap = UIImpactFeedbackGenerator(style: .medium)
    private static let navigate = UIImpactFeedbackGenerator(style: .heavy)
    private static let selection = UISelectionFeedbackGenerator()

    static func pinSelected() {
        pinTap.impactOccurred()
    }

    static func navigationStarted() {
        navigate.impactOccurred()
    }

    static func itemSelected() {
        selection.selectionChanged()
    }
}
