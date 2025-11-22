package response

import (
	"encoding/json"
	"net/http"

	"github.com/V1merX/pr-reviewer-service/internal/api"
)

func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	response := api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode `json:"code"`
			Message string                     `json:"message"`
		}{
			Code:    api.ErrorResponseErrorCode(code),
			Message: message,
		},
	}
	WriteJSON(w, status, response)
}
