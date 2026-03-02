import SwiftUI

struct PulseRing: View {
    @State private var isAnimating = false

    var body: some View {
        ZStack {
            Circle()
                .stroke(Color.accentBlue.opacity(0.3), lineWidth: 2)
                .frame(width: 40, height: 40)
                .scaleEffect(isAnimating ? 2.0 : 1.0)
                .opacity(isAnimating ? 0 : 0.6)

            Circle()
                .fill(Color.accentBlue)
                .frame(width: 14, height: 14)
                .overlay(
                    Circle()
                        .stroke(.white, lineWidth: 2)
                )
        }
        .onAppear {
            withAnimation(
                .easeOut(duration: 2.0)
                .repeatForever(autoreverses: false)
            ) {
                isAnimating = true
            }
        }
    }
}
