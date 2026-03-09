package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/okshelters/shelternav/gateway/client"
	pb "github.com/okshelters/shelternav/gateway/pb"
)

const rpcTimeout = 3 * time.Second

// ShelterHandler holds dependencies for shelter-related HTTP handlers.
type ShelterHandler struct {
	geoClient client.GeoClient
	logger    *slog.Logger
}

// NewShelterHandler creates a handler wired to the geo-service client.
func NewShelterHandler(geoClient client.GeoClient, logger *slog.Logger) *ShelterHandler {
	return &ShelterHandler{
		geoClient: geoClient,
		logger:    logger,
	}
}

// shelterJSON is the JSON response representation of a shelter.
type shelterJSON struct {
	ID        int32   `json:"id"`
	Name      string  `json:"name"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Type      int32   `json:"type"`
	Capacity  int32   `json:"capacity"`
	Occupancy int32   `json:"occupancy"`
	Status    int32   `json:"status"`
	Address   string  `json:"address"`
	DistanceM float64 `json:"distance_m"`
}

type nearestResponse struct {
	Shelters []shelterJSON `json:"shelters"`
	QueryMS  float64       `json:"query_ms"`
}

// HandleFindNearest handles GET /api/v1/shelters/nearest.
func (h *ShelterHandler) HandleFindNearest(w http.ResponseWriter, r *http.Request) {
	lat, err := parseFloat64(r, "lat")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing lat")
		return
	}

	lon, err := parseFloat64(r, "lon")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing lon")
		return
	}

	radiusM, err := parseUint32(r, "radius", 5000)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid radius")
		return
	}

	limit, err := parseUint32(r, "limit", 10)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	if limit > 100 {
		limit = 100
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
	defer cancel()

	resp, err := h.geoClient.FindNearest(ctx, &pb.NearestRequest{
		Lat:     lat,
		Lon:     lon,
		RadiusM: radiusM,
		Limit:   limit,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "handler nearest", slog.String("error", err.Error()))
		WriteJSONError(w, http.StatusBadGateway, "upstream service error")
		return
	}

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

	WriteJSON(w, http.StatusOK, nearestResponse{
		Shelters: shelters,
		QueryMS:  float64(time.Since(start).Microseconds()) / 1000,
	})
}

type routeJSON struct {
	Path             []latLonJSON   `json:"path"`
	TotalDistanceM   float64        `json:"total_distance_m"`
	EstimatedSeconds uint32         `json:"estimated_seconds"`
	Maneuvers        []maneuverJSON `json:"maneuvers"`
	QueryMS          float64        `json:"query_ms"`
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

// HandleGetRoute handles GET /api/v1/route.
func (h *ShelterHandler) HandleGetRoute(w http.ResponseWriter, r *http.Request) {
	fromLat, err := parseFloat64Multi(r, "from_lat", "start_lat")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing from_lat")
		return
	}

	fromLon, err := parseFloat64Multi(r, "from_lon", "start_lon")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing from_lon")
		return
	}

	toLat, err := parseFloat64Multi(r, "to_lat", "end_lat")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing to_lat")
		return
	}

	toLon, err := parseFloat64Multi(r, "to_lon", "end_lon")
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid or missing to_lon")
		return
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
	defer cancel()

	resp, err := h.geoClient.GetRoute(ctx, &pb.RouteRequest{
		StartLat: fromLat,
		StartLon: fromLon,
		EndLat:   toLat,
		EndLon:   toLon,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "handler route", slog.String("error", err.Error()))
		WriteJSONError(w, http.StatusBadGateway, "upstream service error")
		return
	}

	path := make([]latLonJSON, 0, len(resp.GetPath()))
	for _, point := range resp.GetPath() {
		path = append(path, latLonJSON{
			Lat: point.GetLat(),
			Lon: point.GetLon(),
		})
	}

	maneuvers := make([]maneuverJSON, 0, len(resp.GetManeuvers()))
	for _, m := range resp.GetManeuvers() {
		point := latLonJSON{}
		if m.GetPoint() != nil {
			point = latLonJSON{Lat: m.GetPoint().GetLat(), Lon: m.GetPoint().GetLon()}
		}
		maneuvers = append(maneuvers, maneuverJSON{
			Point:       point,
			Instruction: m.GetInstruction(),
			DistanceM:   m.GetDistanceM(),
		})
	}

	WriteJSON(w, http.StatusOK, routeJSON{
		Path:             path,
		TotalDistanceM:   resp.GetTotalDistanceM(),
		EstimatedSeconds: resp.GetEstimatedSeconds(),
		Maneuvers:        maneuvers,
		QueryMS:          float64(time.Since(start).Microseconds()) / 1000,
	})
}

func parseFloat64(r *http.Request, key string) (float64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(raw, 64)
}

func parseFloat64Multi(r *http.Request, keys ...string) (float64, error) {
	for _, key := range keys {
		if raw := r.URL.Query().Get(key); raw != "" {
			return strconv.ParseFloat(raw, 64)
		}
	}
	return 0, strconv.ErrSyntax
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

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteJSONError writes a standard error envelope.
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{
		"error": message,
		"code":  status,
	})
}
