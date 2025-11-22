package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/response"
	"github.com/V1merX/pr-reviewer-service/internal/service"
)

type Handler struct {
	userSvc service.UserService
	prSvc   service.PullRequestService
}

func New(userSvc service.UserService, prSvc service.PullRequestService) *Handler {
	return &Handler{userSvc: userSvc, prSvc: prSvc}
}

func (h *Handler) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	user, err := h.userSvc.SetUserStatus(req.UserID, req.IsActive)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	resp := map[string]interface{}{"user": user}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetUsersGetReview(w http.ResponseWriter, _ *http.Request, params api.GetUsersGetReviewParams) {
	userID := params.UserId
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id parameter is required")
		return
	}

	prs, err := h.prSvc.FindPRsByReviewer(userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		slog.Error("user: get review failed", "error", err)
		return
	}

	var items []api.PullRequestShort
	for _, pr := range prs {
		items = append(items, api.PullRequestShort{
			PullRequestId:   pr.PullRequestId,
			PullRequestName: pr.PullRequestName,
			AuthorId:        pr.AuthorId,
			Status:          api.PullRequestShortStatus(pr.Status),
		})
	}

	resp := map[string]interface{}{"user_id": userID, "pull_requests": items}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) PostUsersDeactivateBatch(w http.ResponseWriter, r *http.Request) {
	var req api.BatchDeactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if len(req.UserIds) == 0 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_ids cannot be empty")
		return
	}

	result, err := h.prSvc.DeactivateUsersAndReassignPRs(req.TeamName, req.UserIds)
	if err != nil {
		if err.Error() == "team not found" {
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Team not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	slog.Info("batch deactivation completed", "team", req.TeamName, "deactivated", result.DeactivatedCount, "reassigned", result.ReassignedCount)
	response.WriteJSON(w, http.StatusOK, result)
}
