package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/okshelters/shelternav/gateway/pb"
	"google.golang.org/grpc"
)

// mockShelterClient implements pb.ShelterServiceClient for testing.
type mockShelterClient struct {
	findNearestFn func(ctx context.Context, in *pb.NearestRequest, opts ...grpc.CallOption) (*pb.NearestResponse, error)
	getRouteFn    func(ctx context.Context, in *pb.RouteRequest, opts ...grpc.CallOption) (*pb.RouteResponse, error)
}

func (m *mockShelterClient) FindNearest(ctx context.Context, in *pb.NearestRequest, opts ...grpc.CallOption) (*pb.NearestResponse, error) {
	return m.findNearestFn(ctx, in, opts...)
}

func (m *mockShelterClient) GetRoute(ctx context.Context, in *pb.RouteRequest, opts ...grpc.CallOption) (*pb.RouteResponse, error) {
	return m.getRouteFn(ctx, in, opts...)
}

// newTestShelterHandler creates a ShelterHandler with an injected mock client.
func newTestShelterHandler(client pb.ShelterServiceClient, logger *slog.Logger) *ShelterHandler {
	return &ShelterHandler{
		geoClient: client,
		logger:    logger,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- FindNearest tests ---

func TestHandleFindNearest_Success(t *testing.T) {
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, in *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			return &pb.NearestResponse{
				Shelters: []*pb.ShelterInfo{
					{
						Id:        1,
						Name:      "Downtown Shelter",
						Lat:       35.4676,
						Lon:       -97.5164,
						Type:      1,
						Capacity:  100,
						Occupancy: 42,
						Status:    1,
						Address:   "123 Main St",
						DistanceM: 250.5,
					},
				},
			}, nil
		},
	}

	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35.4676&lon=-97.5164", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json content-type, got %q", ct)
	}

	var resp nearestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Shelters) != 1 {
		t.Fatalf("expected 1 shelter, got %d", len(resp.Shelters))
	}
	s := resp.Shelters[0]
	if s.ID != 1 {
		t.Errorf("expected shelter ID 1, got %d", s.ID)
	}
	if s.Name != "Downtown Shelter" {
		t.Errorf("expected shelter name 'Downtown Shelter', got %q", s.Name)
	}
	if s.DistanceM != 250.5 {
		t.Errorf("expected distance_m 250.5, got %f", s.DistanceM)
	}
	if resp.QueryMs <= 0 {
		t.Errorf("expected positive query_ms, got %f", resp.QueryMs)
	}
}

func TestHandleFindNearest_MissingLat(t *testing.T) {
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, _ *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			t.Fatal("FindNearest should not be called when lat is missing")
			return nil, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lon=-97.5164", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandleFindNearest_MissingLon(t *testing.T) {
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, _ *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			t.Fatal("FindNearest should not be called when lon is missing")
			return nil, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35.4676", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandleFindNearest_InvalidLat(t *testing.T) {
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, _ *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			t.Fatal("FindNearest should not be called with invalid lat")
			return nil, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=abc&lon=-97.5164", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleFindNearest_DefaultRadiusAndLimit(t *testing.T) {
	var capturedReq *pb.NearestRequest
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, in *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			capturedReq = in
			return &pb.NearestResponse{Shelters: []*pb.ShelterInfo{}}, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35.4676&lon=-97.5164", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if capturedReq == nil {
		t.Fatal("expected gRPC request to be captured")
	}
	if capturedReq.RadiusM != 5000 {
		t.Errorf("expected default radius 5000, got %d", capturedReq.RadiusM)
	}
	if capturedReq.Limit != 10 {
		t.Errorf("expected default limit 10, got %d", capturedReq.Limit)
	}
}

func TestHandleFindNearest_LimitCap(t *testing.T) {
	var capturedReq *pb.NearestRequest
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, in *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			capturedReq = in
			return &pb.NearestResponse{Shelters: []*pb.ShelterInfo{}}, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35.4676&lon=-97.5164&limit=200", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if capturedReq == nil {
		t.Fatal("expected gRPC request to be captured")
	}
	if capturedReq.Limit != 100 {
		t.Errorf("expected limit to be capped at 100, got %d", capturedReq.Limit)
	}
}

func TestHandleFindNearest_UpstreamError(t *testing.T) {
	mock := &mockShelterClient{
		findNearestFn: func(_ context.Context, _ *pb.NearestRequest, _ ...grpc.CallOption) (*pb.NearestResponse, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35.4676&lon=-97.5164", nil)
	rec := httptest.NewRecorder()

	h.HandleFindNearest(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if body["error"] != "upstream service error" {
		t.Errorf("expected 'upstream service error', got %q", body["error"])
	}
}

// --- GetRoute tests ---

func TestHandleGetRoute_Success(t *testing.T) {
	mock := &mockShelterClient{
		getRouteFn: func(_ context.Context, in *pb.RouteRequest, _ ...grpc.CallOption) (*pb.RouteResponse, error) {
			return &pb.RouteResponse{
				Path: []*pb.LatLon{
					{Lat: 35.4676, Lon: -97.5164},
					{Lat: 35.4700, Lon: -97.5100},
				},
				TotalDistanceM:   1500.0,
				EstimatedSeconds: 300,
				Maneuvers: []*pb.Maneuver{
					{
						Point:       &pb.LatLon{Lat: 35.4690, Lon: -97.5130},
						Instruction: "Turn right on 2nd Ave",
						DistanceM:   750.0,
					},
				},
			}, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/route?start_lat=35.4676&start_lon=-97.5164&end_lat=35.4700&end_lon=-97.5100", nil)
	rec := httptest.NewRecorder()

	h.HandleGetRoute(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp routeJSON
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Path) != 2 {
		t.Fatalf("expected 2 path points, got %d", len(resp.Path))
	}
	if resp.TotalDistanceM != 1500.0 {
		t.Errorf("expected total_distance_m 1500.0, got %f", resp.TotalDistanceM)
	}
	if resp.EstimatedSeconds != 300 {
		t.Errorf("expected estimated_seconds 300, got %d", resp.EstimatedSeconds)
	}
	if len(resp.Maneuvers) != 1 {
		t.Fatalf("expected 1 maneuver, got %d", len(resp.Maneuvers))
	}
	if resp.Maneuvers[0].Instruction != "Turn right on 2nd Ave" {
		t.Errorf("unexpected maneuver instruction: %q", resp.Maneuvers[0].Instruction)
	}
	if resp.QueryMs <= 0 {
		t.Errorf("expected positive query_ms, got %f", resp.QueryMs)
	}
}

func TestHandleGetRoute_MissingParams(t *testing.T) {
	mock := &mockShelterClient{
		getRouteFn: func(_ context.Context, _ *pb.RouteRequest, _ ...grpc.CallOption) (*pb.RouteResponse, error) {
			t.Fatal("GetRoute should not be called with missing params")
			return nil, nil
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	tests := []struct {
		name  string
		query string
	}{
		{"missing start_lat", "start_lon=-97.5164&end_lat=35.47&end_lon=-97.51"},
		{"missing start_lon", "start_lat=35.4676&end_lat=35.47&end_lon=-97.51"},
		{"missing end_lat", "start_lat=35.4676&start_lon=-97.5164&end_lon=-97.51"},
		{"missing end_lon", "start_lat=35.4676&start_lon=-97.5164&end_lat=35.47"},
		{"all missing", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/route?"+tc.query, nil)
			rec := httptest.NewRecorder()

			h.HandleGetRoute(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", rec.Code)
			}
		})
	}
}

func TestHandleGetRoute_UpstreamError(t *testing.T) {
	mock := &mockShelterClient{
		getRouteFn: func(_ context.Context, _ *pb.RouteRequest, _ ...grpc.CallOption) (*pb.RouteResponse, error) {
			return nil, fmt.Errorf("upstream unavailable")
		},
	}
	h := newTestShelterHandler(mock, discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/route?start_lat=35.4676&start_lon=-97.5164&end_lat=35.47&end_lon=-97.51", nil)
	rec := httptest.NewRecorder()

	h.HandleGetRoute(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if body["error"] != "upstream service error" {
		t.Errorf("expected 'upstream service error', got %q", body["error"])
	}
}
