use std::sync::Arc;
use tokio::sync::RwLock;
use tonic::{Request, Response, Status};

use crate::rtree::{QueryConstraints, ShelterIndex};

/// Generated protobuf / tonic code.
pub mod pb {
    tonic::include_proto!("shelternav.v1");
}

use pb::shelter_service_server::ShelterService;
use pb::{NearestRequest, NearestResponse, RouteRequest, RouteResponse, ShelterInfo};

/// gRPC service implementation backed by an in-memory R-tree.
pub struct ShelterServiceImpl {
    /// Shared shelter index guarded by an async read/write lock.
    pub index: Arc<RwLock<ShelterIndex>>,
}

#[tonic::async_trait]
impl ShelterService for ShelterServiceImpl {
    /// Find the nearest shelters to a given (lat, lon) within a radius.
    ///
    /// Hot path: acquires only a read lock on the index, so concurrent
    /// FindNearest calls never block each other.
    async fn find_nearest(
        &self,
        request: Request<NearestRequest>,
    ) -> Result<Response<NearestResponse>, Status> {
        let req = request.into_inner();

        // Validate inputs.
        if !(-90.0..=90.0).contains(&req.lat) || !(-180.0..=180.0).contains(&req.lon) {
            return Err(Status::invalid_argument(
                "lat must be in [-90, 90] and lon in [-180, 180]",
            ));
        }

        let radius_m = if req.radius_m == 0 {
            5_000.0 // default 5 km
        } else {
            f64::from(req.radius_m)
        };

        let limit = if req.limit == 0 {
            10 // default limit
        } else {
            req.limit.min(100) as usize
        };

        let index = self.index.read().await;
        let results = index.find_nearest(
            req.lat,
            req.lon,
            QueryConstraints {
                radius_m,
                limit,
                allowed_types: req.types.as_slice(),
                open_only: true,
            },
        );

        let mut shelters = Vec::with_capacity(results.len());
        for (point, distance_m) in results {
            shelters.push(ShelterInfo {
                id: point.id,
                name: point.name.to_string(),
                lat: point.lat,
                lon: point.lon,
                r#type: point.shelter_type,
                capacity: point.capacity,
                occupancy: point.occupancy,
                status: point.status,
                address: point.address.to_string(),
                distance_m,
            });
        }

        Ok(Response::new(NearestResponse { shelters }))
    }

    /// Compute a walking route between two points.
    ///
    /// Not yet implemented -- the routing engine lives in the C client core
    /// library. This endpoint will proxy to that engine or use a server-side
    /// A* implementation once the road graph is loaded.
    async fn get_route(
        &self,
        _request: Request<RouteRequest>,
    ) -> Result<Response<RouteResponse>, Status> {
        Err(Status::unimplemented(
            "GetRoute is not yet implemented on the server; use the client-side A* engine",
        ))
    }
}
