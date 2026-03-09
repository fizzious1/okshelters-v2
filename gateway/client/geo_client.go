package client

import (
	"context"
	"fmt"

	pb "github.com/okshelters/shelternav/gateway/pb"
	"google.golang.org/grpc"
)

// GeoClient defines the gRPC calls the gateway uses from geo-service.
type GeoClient interface {
	FindNearest(ctx context.Context, req *pb.NearestRequest) (*pb.NearestResponse, error)
	GetRoute(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error)
}

// GRPCGeoClient wraps the generated protobuf client with typed error wrapping.
type GRPCGeoClient struct {
	client pb.ShelterServiceClient
}

// NewGRPCGeoClient creates a geo client from a shared grpc connection.
func NewGRPCGeoClient(conn grpc.ClientConnInterface) *GRPCGeoClient {
	return &GRPCGeoClient{client: pb.NewShelterServiceClient(conn)}
}

// FindNearest forwards the call to geo-service.
func (c *GRPCGeoClient) FindNearest(ctx context.Context, req *pb.NearestRequest) (*pb.NearestResponse, error) {
	resp, err := c.client.FindNearest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("geo client find nearest: %w", err)
	}
	return resp, nil
}

// GetRoute forwards the call to geo-service.
func (c *GRPCGeoClient) GetRoute(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error) {
	resp, err := c.client.GetRoute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("geo client get route: %w", err)
	}
	return resp, nil
}
