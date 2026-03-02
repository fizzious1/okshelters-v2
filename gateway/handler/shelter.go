package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	pb "github.com/okshelters/shelternav/gateway/pb"
	"google.golang.org/grpc"
)

// ShelterHandler holds dependencies for shelter-related HTTP handlers.
type ShelterHandler struct {
	geoClient pb.ShelterServiceClient
	logger    *slog.Logger
}

// NewShelterHandler creates a handler wired to the geo-service gRPC connection.
func NewShelterHandler(conn grpc.ClientConnInterface, logger *slog.Logger) *ShelterHandler {
	return &ShelterHandler{
		geoClient: pb.NewShelterServiceClient(conn),
		logger:    logger,
	}
}

// shelterJSON is the JSON response representation of a shelter.
// Kept separate from protobuf to control the public API surface.
type shelterJSON struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Type       int32   `json:"type"`
	Capacity   int32   `json:"capacity"`
	Occupancy  int32   `json:"occupancy"`
	Status     int32   `json:"status"`
	Address    string  `json:"address"`
	DistanceM  float64 `json:"distance_m"`
}

// nearestResponse is the top-level JSON envelope for find-nearest results.
type nearestResponse struct {
	Shelters []shelterJSON `json:"shelters"`
	QueryMs  float64       `json:"query_ms"`
}

// HandleFindNearest parses lat/lon/radius/limit from query params, calls
// the geo-service FindNearest RPC, and returns a JSON response.
func (h *ShelterHandler) HandleFindNearest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	lat, err := parseFloat64(r, "lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'lat' parameter")
		return
	}

	lon, err := parseFloat64(r, "lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'lon' parameter")
		return
	}

	radiusM, err := parseUint32(r, "radius", 5000)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'radius' parameter")
		return
	}

	limit, err := parseUint32(r, "limit", 10)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'limit' parameter")
		return
	}

	// Cap limit to prevent abuse.
	if limit > 100 {
		limit = 100
	}

	start := time.Now()

	// Propagate request context with a tight deadline for the geo-service call.
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := h.geoClient.FindNearest(rpcCtx, &pb.NearestRequest{
		Lat:      lat,
		Lon:      lon,
		RadiusM:  radiusM,
		Limit:    limit,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "geo-service FindNearest failed",
			slog.String("error", err.Error()),
			slog.Float64("lat", lat),
			slog.Float64("lon", lon),
		)
		writeError(w, http.StatusBadGateway, "upstream service error")
		return
	}

	elapsed := time.Since(start)

	shelters := make([]shelterJSON, 0, len(resp.GetShelters()))
	for _, s := range resp.GetShelters() {
		shelters = append(shelters, shelterJSON{
			ID:        s.GetId(),
			Name:      s.GetName(),
			Lat:       s.GetLat(),
			Lon:       s.GetLon(),
			Type:      s.GetType(),
			Capacity:  s.GetCapacity(),
			Occupancy: s.GetOccupancy(),
			Status:    s.GetStatus(),
			Address:   s.GetAddress(),
			DistanceM: s.GetDistanceM(),
		})
	}

	writeJSON(w, http.StatusOK, nearestResponse{
		Shelters: shelters,
		QueryMs:  float64(elapsed.Microseconds()) / 1000.0,
	})
}

// routeJSON is the JSON response representation of a route.
type routeJSON struct {
	Path             []latLonJSON   `json:"path"`
	TotalDistanceM   float64        `json:"total_distance_m"`
	EstimatedSeconds uint32         `json:"estimated_seconds"`
	Maneuvers        []maneuverJSON `json:"maneuvers"`
	QueryMs          float64        `json:"query_ms"`
}

type latLonJSON struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type maneuverJSON struct {
	Point       latLonJSON `json:"point"`
	Instruction string     `json:"instruction"`
	DistanceM   float64    `json:"distance_m"`
}

// HandleGetRoute parses start/end coordinates, calls the geo-service GetRoute
// RPC, and returns a JSON response.
func (h *ShelterHandler) HandleGetRoute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	startLat, err := parseFloat64(r, "start_lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'start_lat' parameter")
		return
	}

	startLon, err := parseFloat64(r, "start_lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'start_lon' parameter")
		return
	}

	endLat, err := parseFloat64(r, "end_lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'end_lat' parameter")
		return
	}

	endLon, err := parseFloat64(r, "end_lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing 'end_lon' parameter")
		return
	}

	start := time.Now()

	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := h.geoClient.GetRoute(rpcCtx, &pb.RouteRequest{
		StartLat: startLat,
		StartLon: startLon,
		EndLat:   endLat,
		EndLon:   endLon,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "geo-service GetRoute failed",
			slog.String("error", err.Error()),
			slog.Float64("start_lat", startLat),
			slog.Float64("start_lon", startLon),
			slog.Float64("end_lat", endLat),
			slog.Float64("end_lon", endLon),
		)
		writeError(w, http.StatusBadGateway, "upstream service error")
		return
	}

	elapsed := time.Since(start)

	path := make([]latLonJSON, 0, len(resp.GetPath()))
	for _, p := range resp.GetPath() {
		path = append(path, latLonJSON{Lat: p.GetLat(), Lon: p.GetLon()})
	}

	maneuvers := make([]maneuverJSON, 0, len(resp.GetManeuvers()))
	for _, m := range resp.GetManeuvers() {
		pt := latLonJSON{}
		if m.GetPoint() != nil {
			pt = latLonJSON{Lat: m.GetPoint().GetLat(), Lon: m.GetPoint().GetLon()}
		}
		maneuvers = append(maneuvers, maneuverJSON{
			Point:       pt,
			Instruction: m.GetInstruction(),
			DistanceM:   m.GetDistanceM(),
		})
	}

	writeJSON(w, http.StatusOK, routeJSON{
		Path:             path,
		TotalDistanceM:   resp.GetTotalDistanceM(),
		EstimatedSeconds: resp.GetEstimatedSeconds(),
		Maneuvers:        maneuvers,
		QueryMs:          float64(elapsed.Microseconds()) / 1000.0,
	})
}

// --- helpers ---

func parseFloat64(r *http.Request, key string) (float64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(raw, 64)
}

func parseUint32(r *http.Request, key string, defaultVal uint32) (uint32, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal, nil
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
