package team

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/response"
	"github.com/V1merX/pr-reviewer-service/internal/service"
)

type Handler struct {
	svc service.TeamService
}

func New(svc service.TeamService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	var req api.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.svc.AddTeam(&req); err != nil {
		if err.Error() == "team already exists" {
			response.WriteError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		} else {
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			slog.Error("team: add failed", "error", err)
		}
		return
	}

	resp := map[string]interface{}{"team": req}
	response.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) GetTeamGet(w http.ResponseWriter, _ *http.Request, params api.GetTeamGetParams) {
	teamName := params.TeamName
	if teamName == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name parameter is required")
		return
	}

	team, err := h.svc.GetTeamByName(teamName)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Team not found")
		return
	}
	response.WriteJSON(w, http.StatusOK, team)
}
