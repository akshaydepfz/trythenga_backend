package helper

import (
	"encoding/json"
	"net/http"
)

func JSONResponse(w http.ResponseWriter, data any) {
	JSONResponseWithStatus(w, data, http.StatusOK)
}

func JSONResponseWithStatus(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(data)
}

func JSONError(w http.ResponseWriter, message string, statusCode int) {
	JSONResponseWithStatus(w, map[string]string{"error": message}, statusCode)
}
