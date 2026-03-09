package handler

import "net/http"

// HandleHealthz reports process liveness.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
