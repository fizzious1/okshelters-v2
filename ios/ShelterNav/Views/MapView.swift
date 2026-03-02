import SwiftUI

// MapLibre Native iOS wrapper
// Wraps MGLMapView via UIViewRepresentable
struct MapView: UIViewRepresentable {

    func makeUIView(context: Context) -> UIView {
        // TODO: Replace with MGLMapView from MapLibre Native iOS SDK
        // let mapView = MGLMapView(frame: .zero, styleURL: styleURL)
        // mapView.delegate = context.coordinator
        // mapView.showsUserLocation = true
        let placeholder = UIView()
        placeholder.backgroundColor = UIColor(Color.bgPrimary)
        return placeholder
    }

    func updateUIView(_ uiView: UIView, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    class Coordinator: NSObject {
        // TODO: Implement MGLMapViewDelegate
    }
}
