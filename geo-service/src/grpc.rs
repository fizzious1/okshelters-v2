use std::sync::Arc;
use tokio::sync::RwLock;
use tonic::{Request, Response, Status};

use crate::rtree::ShelterIndex;

/// Generated protobuf / tonic code.
pub mod pb {
    tonic::include_proto!("shelternav.v1");
}

use pb::shelter_service_server::ShelterService;
use pb::{NearestRequest, NearestResponse, RouteRequest, RouteResponse, ShelterInfo};

/// gRPC service implementation backed by an in-memory R-tree.
pub struct ShelterServiceImpl {
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
            20 // default limit
        } else {
            req.limit as usize
        };

        let index = self.index.read().await;
        let results = index.find_nearest(req.lat, req.lon, radius_m, limit);

        let shelters: Vec<ShelterInfo> = results
            .into_iter()
            .filter(|(p, _dist)| {
                // If caller specified type filters, enforce them.
                req.types.is_empty() || req.types.contains(&p.shelter_type)
            })
            .map(|(p, dist)| ShelterInfo {
                id: p.id,
                name: p.name,
                lat: p.lat,
                lon: p.lon,
                r#type: p.shelter_type,
                capacity: p.capacity,
                occupancy: p.occupancy,
                status: p.status,
                address: p.address,
                distance_m: dist,
            })
            .collect();

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
