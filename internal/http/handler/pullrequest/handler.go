package pullrequest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/response"
	"github.com/V1merX/pr-reviewer-service/internal/service"
)

type Handler struct {
	prSvc service.PullRequestService
}

func New(prSvc service.PullRequestService) *Handler {
	return &Handler{prSvc: prSvc}
}

func (h *Handler) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	prExists, err := h.prSvc.FindPRByID(req.PullRequestID)
	if err == nil && prExists != nil {
		response.WriteError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
		return
	}
	if err != nil {
		slog.Debug("check PR exists failed", "err", err)
	}

	pr := &api.PullRequest{
		PullRequestId:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorId:        req.AuthorID,
	}

	if err := h.prSvc.CreatePR(pr); err != nil {
		if err.Error() == "author not found" || err.Error() == "author has no team" {
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Author or team not found")
		} else {
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			slog.Error("pr: create failed", "error", err)
		}
		return
	}

	response.WriteJSON(w, http.StatusCreated, map[string]interface{}{"pr": pr})
}

func (h *Handler) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.prSvc.MergePR(req.PullRequestID)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{"pr": pr})
}

func (h *Handler) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pr, newReviewer, err := h.prSvc.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err.Error() {
		case "PR not found":
			response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		case "cannot reassign on merged PR":
			response.WriteError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		case "reviewer is not assigned to this PR":
			response.WriteError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		case "no active replacement candidate in team":
			response.WriteError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		default:
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			slog.Error("pr: reassign failed", "error", err)
		}
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]interface{}{"pr": pr, "replaced_by": *newReviewer})
}

func (h *Handler) GetStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := h.prSvc.GetStatistics()
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, stats)
}
