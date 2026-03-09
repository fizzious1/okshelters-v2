package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/okshelters/shelternav/gateway/pb"
)

type fakeGeoClient struct {
	findNearestFn func(ctx context.Context, req *pb.NearestRequest) (*pb.NearestResponse, error)
	getRouteFn    func(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error)
}

func (f fakeGeoClient) FindNearest(ctx context.Context, req *pb.NearestRequest) (*pb.NearestResponse, error) {
	if f.findNearestFn == nil {
		return nil, errors.New("find nearest not implemented")
	}
	return f.findNearestFn(ctx, req)
}

func (f fakeGeoClient) GetRoute(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error) {
	if f.getRouteFn == nil {
		return nil, errors.New("get route not implemented")
	}
	return f.getRouteFn(ctx, req)
}

func TestHandleFindNearestSuccess(t *testing.T) {
	t.Parallel()

	var gotDeadline time.Time
	geo := fakeGeoClient{
		findNearestFn: func(ctx context.Context, req *pb.NearestRequest) (*pb.NearestResponse, error) {
			if req.Lat != 31.77 || req.Lon != 35.21 || req.RadiusM != 2500 || req.Limit != 5 {
				t.Fatalf("unexpected request: %+v", req)
			}

			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatalf("expected deadline on rpc context")
			}
			gotDeadline = deadline

			return &pb.NearestResponse{
				Shelters: []*pb.ShelterInfo{{
					Id:        42,
					Name:      "Shelter A",
					Lat:       31.7701,
					Lon:       35.2102,
					Type:      1,
					Capacity:  120,
					Occupancy: 35,
					Status:    1,
					Address:   "Somewhere",
					DistanceM: 87.4,
				}},
			}, nil
		},
	}

	h := NewShelterHandler(geo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=31.77&lon=35.21&radius=2500&limit=5", nil)
	rr := httptest.NewRecorder()

	h.HandleFindNearest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}

	var body struct {
		Shelters []struct {
			ID   int32  `json:"id"`
			Name string `json:"name"`
		} `json:"shelters"`
		QueryMS float64 `json:"query_ms"`
	}
	decodeJSON(t, rr.Body.Bytes(), &body)

	if len(body.Shelters) != 1 || body.Shelters[0].ID != 42 || body.Shelters[0].Name != "Shelter A" {
		t.Fatalf("unexpected shelters payload: %+v", body.Shelters)
	}
	if body.QueryMS < 0 {
		t.Fatalf("query_ms must be non-negative")
	}

	remaining := time.Until(gotDeadline)
	if remaining <= 0 || remaining > 3*time.Second {
		t.Fatalf("unexpected deadline window remaining: %v", remaining)
	}
}

func TestHandleFindNearestBadLat(t *testing.T) {
	t.Parallel()

	h := NewShelterHandler(fakeGeoClient{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=bad&lon=35.21", nil)
	rr := httptest.NewRecorder()

	h.HandleFindNearest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var errBody struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	decodeJSON(t, rr.Body.Bytes(), &errBody)
	if errBody.Code != http.StatusBadRequest || errBody.Error == "" {
		t.Fatalf("unexpected error payload: %+v", errBody)
	}
}

func TestHandleFindNearestUpstreamError(t *testing.T) {
	t.Parallel()

	geo := fakeGeoClient{
		findNearestFn: func(context.Context, *pb.NearestRequest) (*pb.NearestResponse, error) {
			return nil, errors.New("boom")
		},
	}

	h := NewShelterHandler(geo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=31.77&lon=35.21", nil)
	rr := httptest.NewRecorder()

	h.HandleFindNearest(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rr.Code)
	}
}

func TestHandleGetRouteSuccess(t *testing.T) {
	t.Parallel()

	geo := fakeGeoClient{
		getRouteFn: func(_ context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error) {
			if req.StartLat != 31.7 || req.StartLon != 35.2 || req.EndLat != 31.8 || req.EndLon != 35.3 {
				t.Fatalf("unexpected route request: %+v", req)
			}
			return &pb.RouteResponse{
				Path:             []*pb.LatLon{{Lat: 31.7, Lon: 35.2}, {Lat: 31.8, Lon: 35.3}},
				TotalDistanceM:   1500,
				EstimatedSeconds: 720,
				Maneuvers: []*pb.Maneuver{{
					Point:       &pb.LatLon{Lat: 31.71, Lon: 35.22},
					Instruction: "Head north",
					DistanceM:   100,
				}},
			}, nil
		},
	}

	h := NewShelterHandler(geo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/route?from_lat=31.7&from_lon=35.2&to_lat=31.8&to_lon=35.3", nil)
	rr := httptest.NewRecorder()

	h.HandleGetRoute(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body struct {
		Path []struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"path"`
		TotalDistanceM   float64 `json:"total_distance_m"`
		EstimatedSeconds uint32  `json:"estimated_seconds"`
	}
	decodeJSON(t, rr.Body.Bytes(), &body)

	if len(body.Path) != 2 {
		t.Fatalf("unexpected path points: %+v", body.Path)
	}
	if body.TotalDistanceM != 1500 || body.EstimatedSeconds != 720 {
		t.Fatalf("unexpected route summary: %+v", body)
	}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("failed to decode json: %v\nbody=%s", err, body)
	}
}
