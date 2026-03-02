import CoreLocation
import Foundation

// FFI bridge to libshelternav C library.
// Links against libshelternav via bridging header.
struct RouteResult {
    let path: [CLLocationCoordinate2D]
    let totalDistanceM: Double
    let estimatedSeconds: Int
}

final class ShelterNavBridge {
    static let shared = ShelterNavBridge()

    private let queue = DispatchQueue(label: "com.okshelters.ios.bridge", qos: .userInitiated)
    private var tree: UnsafeMutablePointer<SN_KDNode>?
    private var isDatabaseOpen = false
    private var hasSynced = false

    init() {
        queue.sync {
            _ = openDatabaseIfNeeded()
            _ = syncTree(force: true)
        }
    }

    deinit {
        queue.sync {
            if let tree {
                sn_kdtree_destroy(tree)
                self.tree = nil
            }
            if isDatabaseOpen {
                _ = sn_db_close()
                isDatabaseOpen = false
            }
        }
    }

    func findNearest(lat: Double, lon: Double, radiusM: UInt32, limit: UInt32) async -> [ShelterInfo] {
        await performOnBridgeQueue {
            guard radiusM > 0, limit > 0 else { return [] }

            if !hasSynced {
                _ = syncTree(force: true)
            }
            guard let tree else { return [] }

            var shelters = Array(repeating: SN_Shelter(), count: Int(limit))
            let count = shelters.withUnsafeMutableBufferPointer { buffer -> Int32 in
                guard let baseAddress = buffer.baseAddress else { return Int32(SN_ERR_INVALID_ARG) }
                return sn_find_nearest(
                    tree,
                    lat,
                    lon,
                    Double(radiusM),
                    baseAddress,
                    Int32(limit)
                )
            }

            guard count > 0 else { return [] }

            let validCount = min(Int(count), shelters.count)
            return shelters.prefix(validCount)
                .map { mapShelter($0, originLat: lat, originLon: lon) }
                .sorted { $0.distanceM < $1.distanceM }
        }
    }

    func getRoute(startLat: Double, startLon: Double, endLat: Double, endLon: Double) async -> RouteResult {
        await performOnBridgeQueue {
            var maneuvers = Array(repeating: SN_Maneuver(), count: 256)
            var pathLength = Int32(maneuvers.count)

            let start = SN_LatLon(lat: startLat, lon: startLon)
            let end = SN_LatLon(lat: endLat, lon: endLon)
            let result = maneuvers.withUnsafeMutableBufferPointer { buffer -> Int32 in
                guard let baseAddress = buffer.baseAddress else { return Int32(SN_ERR_INVALID_ARG) }
                return sn_route_astar(nil, start, end, baseAddress, &pathLength)
            }

            if result == SN_OK, pathLength > 0 {
                let count = min(Int(pathLength), maneuvers.count)
                let slice = maneuvers.prefix(count)
                let path = slice.map {
                    CLLocationCoordinate2D(
                        latitude: $0.point.lat,
                        longitude: $0.point.lon
                    )
                }
                let totalDistance = slice.reduce(0.0) { partial, maneuver in
                    partial + max(0, maneuver.distance_m)
                }

                return RouteResult(
                    path: path,
                    totalDistanceM: totalDistance,
                    estimatedSeconds: estimateSeconds(distanceM: totalDistance)
                )
            }

            return fallbackRoute(
                startLat: startLat,
                startLon: startLon,
                endLat: endLat,
                endLon: endLon
            )
        }
    }

    func syncDatabase() async {
        _ = await performOnBridgeQueue { syncTree(force: true) }
    }

    private func databasePath() -> String {
        let directoryURL = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        return directoryURL.appendingPathComponent("shelternav.db").path
    }

    private func openDatabaseIfNeeded() -> Bool {
        guard !isDatabaseOpen else { return true }

        let openResult = databasePath().withCString { path in
            sn_db_open(path)
        }
        if openResult == SN_OK {
            isDatabaseOpen = true
            return true
        }
        return false
    }

    private func syncTree(force: Bool) -> Int32 {
        guard openDatabaseIfNeeded() else { return Int32(SN_ERR_DB_OPEN) }
        if !force, hasSynced { return SN_OK }

        if let tree {
            sn_kdtree_destroy(tree)
            self.tree = nil
        }

        let syncResult = sn_db_sync(&tree)
        hasSynced = syncResult >= 0
        return Int32(syncResult)
    }

    private func performOnBridgeQueue<T>(_ work: @escaping () -> T) async -> T {
        await withCheckedContinuation { continuation in
            queue.async {
                continuation.resume(returning: work())
            }
        }
    }

    private func mapShelter(_ cShelter: SN_Shelter, originLat: Double, originLon: Double) -> ShelterInfo {
        let status = status(for: cShelter.status)
        let capacity = Int(cShelter.capacity)

        return ShelterInfo(
            id: cShelter.id,
            name: decodeCString(cShelter.name),
            lat: cShelter.lat,
            lon: cShelter.lon,
            type: .emergency,
            capacity: capacity,
            occupancy: estimatedOccupancy(for: status, capacity: capacity),
            status: status,
            address: decodeCString(cShelter.address),
            phone: nil,
            distanceM: sn_haversine(originLat, originLon, cShelter.lat, cShelter.lon)
        )
    }

    private func status(for rawValue: UInt8) -> ShelterStatus {
        switch rawValue {
        case 1: return .open
        case 2: return .full
        default: return .closed
        }
    }

    private func estimatedOccupancy(for status: ShelterStatus, capacity: Int) -> Int {
        guard capacity > 0 else { return 0 }

        switch status {
        case .open:
            return max(1, Int(Double(capacity) * 0.6))
        case .full:
            return capacity
        case .closed:
            return 0
        }
    }

    private func decodeCString<T>(_ tuple: T) -> String {
        var value = tuple
        return withUnsafePointer(to: &value) { pointer in
            pointer.withMemoryRebound(to: CChar.self, capacity: MemoryLayout<T>.size) { cString in
                String(cString: cString)
            }
        }
    }

    private func estimateSeconds(distanceM: Double) -> Int {
        let walkingMetersPerSecond = 1.4
        return max(0, Int(distanceM / walkingMetersPerSecond))
    }

    private func fallbackRoute(startLat: Double, startLon: Double, endLat: Double, endLon: Double) -> RouteResult {
        let start = CLLocationCoordinate2D(latitude: startLat, longitude: startLon)
        let end = CLLocationCoordinate2D(latitude: endLat, longitude: endLon)
        let distanceM = sn_haversine(startLat, startLon, endLat, endLon)

        return RouteResult(
            path: [start, end],
            totalDistanceM: distanceM,
            estimatedSeconds: estimateSeconds(distanceM: distanceM)
        )
    }
}
