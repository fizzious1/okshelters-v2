import CoreLocation

// FFI bridge to libshelternav C library
// Links against libshelternav.dylib via bridging header

struct RouteResult {
    let path: [CLLocationCoordinate2D]
    let totalDistanceM: Double
    let estimatedSeconds: Int
}

final class ShelterNavBridge {

    init() {
        // TODO: Call sn_db_open() to initialize local SQLite cache
    }

    deinit {
        // TODO: Call sn_db_close()
    }

    func findNearest(lat: Double, lon: Double, radiusM: UInt32, limit: UInt32) async -> [ShelterInfo] {
        // TODO: Call sn_find_nearest() via C interop
        // let buffer = UnsafeMutablePointer<SN_Shelter>.allocate(capacity: Int(limit))
        // defer { buffer.deallocate() }
        // let count = sn_find_nearest(tree, lat, lon, Double(radiusM), buffer, Int32(limit))
        // return (0..<Int(count)).map { ShelterInfo(cShelter: buffer[$0]) }
        return []
    }

    func getRoute(startLat: Double, startLon: Double, endLat: Double, endLon: Double) async -> RouteResult {
        // TODO: Call sn_route_astar() via C interop
        return RouteResult(path: [], totalDistanceM: 0, estimatedSeconds: 0)
    }

    func syncDatabase() async {
        // TODO: Call sn_db_sync() to refresh local cache from network
    }
}
