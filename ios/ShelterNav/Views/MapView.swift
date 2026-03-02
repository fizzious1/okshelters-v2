import SwiftUI
#if canImport(MapLibre)
import MapLibre
#endif

// MapLibre Native iOS wrapper
// Wraps MGLMapView via UIViewRepresentable
struct MapView: UIViewRepresentable {
    @EnvironmentObject private var mapViewModel: MapViewModel
    @EnvironmentObject private var navigationViewModel: NavigationViewModel

#if canImport(MapLibre)
    func makeUIView(context: Context) -> MGLMapView {
        let styleURL = Bundle.main.url(forResource: "map-style", withExtension: "json")
        let mapView = MGLMapView(frame: .zero, styleURL: styleURL)
        mapView.delegate = context.coordinator
        mapView.showsUserLocation = true
        mapView.automaticallyAdjustsContentInset = false
        mapView.logoView.isHidden = true
        mapView.attributionButton.isHidden = true
        MGLOfflineStorage.shared.maximumAmbientCacheSize = 200 * 1_024 * 1_024
        return mapView
    }

    func updateUIView(_ mapView: MGLMapView, context: Context) {
        context.coordinator.refreshShelterLayer(
            on: mapView,
            shelters: mapViewModel.nearbyShelters
        )
        context.coordinator.refreshRouteLayer(
            on: mapView,
            coordinates: navigationViewModel.routeCoordinates
        )

        guard context.coordinator.lastRecenterRequestID != mapViewModel.recenterRequestID else {
            return
        }
        context.coordinator.lastRecenterRequestID = mapViewModel.recenterRequestID

        if let userLocation = mapViewModel.userLocation {
            mapView.setCenter(userLocation, zoomLevel: max(mapView.zoomLevel, 14), animated: true)
        }
    }
#else
    func makeUIView(context: Context) -> UIView {
        let placeholder = UIView()
        placeholder.backgroundColor = UIColor(Color.bgPrimary)
        return placeholder
    }

    func updateUIView(_ uiView: UIView, context: Context) {}
#endif

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    final class Coordinator: NSObject {
        var lastRecenterRequestID = UUID()

#if canImport(MapLibre)
        private let shelterSourceID = "shelter-source"
        private let shelterLayerID = "shelter-layer"
        private let routeSourceID = "route-source"
        private let routeLayerID = "route-layer"

        func refreshShelterLayer(on mapView: MGLMapView, shelters: [ShelterInfo]) {
            guard let style = mapView.style else { return }
            configureStyleIfNeeded(style: style)

            guard let source = style.source(withIdentifier: shelterSourceID) as? MGLShapeSource else {
                return
            }

            let features = shelters.map { shelter -> MGLPointFeature in
                let feature = MGLPointFeature()
                feature.coordinate = CLLocationCoordinate2D(latitude: shelter.lat, longitude: shelter.lon)
                feature.attributes = [
                    "name": shelter.name,
                    "status": shelter.status.rawValue
                ]
                return feature
            }

            source.shape = features.isEmpty ? nil : MGLShapeCollectionFeature(shapes: features)
        }

        func refreshRouteLayer(on mapView: MGLMapView, coordinates: [CLLocationCoordinate2D]) {
            guard let style = mapView.style else { return }
            configureStyleIfNeeded(style: style)

            guard let source = style.source(withIdentifier: routeSourceID) as? MGLShapeSource else {
                return
            }
            guard coordinates.count > 1 else {
                source.shape = nil
                return
            }

            var routeCoordinates = coordinates
            source.shape = MGLPolylineFeature(
                coordinates: &routeCoordinates,
                count: UInt(routeCoordinates.count)
            )
        }

        private func configureStyleIfNeeded(style: MGLStyle) {
            if style.source(withIdentifier: shelterSourceID) == nil {
                let shelterSource = MGLShapeSource(
                    identifier: shelterSourceID,
                    shape: nil,
                    options: [
                        .clustered: true,
                        .clusterRadius: 50
                    ]
                )
                style.addSource(shelterSource)
            }

            if style.layer(withIdentifier: shelterLayerID) == nil,
               let shelterSource = style.source(withIdentifier: shelterSourceID) as? MGLSource {
                let shelterLayer = MGLCircleStyleLayer(identifier: shelterLayerID, source: shelterSource)
                shelterLayer.circleColor = NSExpression(forConstantValue: UIColor(Color.accentBlue))
                shelterLayer.circleRadius = NSExpression(forConstantValue: 6)
                shelterLayer.circleOpacity = NSExpression(forConstantValue: 0.9)
                style.addLayer(shelterLayer)
            }

            if style.source(withIdentifier: routeSourceID) == nil {
                let routeSource = MGLShapeSource(identifier: routeSourceID, shape: nil, options: nil)
                style.addSource(routeSource)
            }

            if style.layer(withIdentifier: routeLayerID) == nil,
               let routeSource = style.source(withIdentifier: routeSourceID) as? MGLSource {
                let routeLayer = MGLLineStyleLayer(identifier: routeLayerID, source: routeSource)
                routeLayer.lineColor = NSExpression(forConstantValue: UIColor(Color.accentBlue))
                routeLayer.lineWidth = NSExpression(forConstantValue: 4)
                routeLayer.lineOpacity = NSExpression(forConstantValue: 0.95)
                routeLayer.lineJoin = NSExpression(forConstantValue: "round")
                routeLayer.lineCap = NSExpression(forConstantValue: "round")
                style.addLayer(routeLayer)
            }
        }
#endif
    }
}

#if canImport(MapLibre)
extension MapView.Coordinator: MGLMapViewDelegate {
    func mapView(_ mapView: MGLMapView, didFinishLoading style: MGLStyle) {
        configureStyleIfNeeded(style: style)
    }
}
#endif
