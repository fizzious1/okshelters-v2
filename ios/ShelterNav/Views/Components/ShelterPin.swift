import SwiftUI

struct ShelterPin: View {
    let status: ShelterStatus

    var body: some View {
        VStack(spacing: 0) {
            Circle()
                .fill(status.color)
                .frame(width: 28, height: 28)
                .overlay(
                    Circle()
                        .stroke(.white, lineWidth: 2)
                )
                .shadow(color: .black.opacity(0.3), radius: 4, y: 2)

            Triangle()
                .fill(status.color)
                .frame(width: 10, height: 6)
                .offset(y: -1)
        }
    }
}

private struct Triangle: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.move(to: CGPoint(x: rect.midX, y: rect.maxY))
        path.addLine(to: CGPoint(x: rect.minX, y: rect.minY))
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.minY))
        path.closeSubpath()
        return path
    }
}
