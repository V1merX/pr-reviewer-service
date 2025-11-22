package handler

import (
	"net/http"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/pullrequest"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/team"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler/user"
	"github.com/V1merX/pr-reviewer-service/internal/service"
	"github.com/go-chi/chi/v5"
)

type ServerHandler struct {
	team *team.Handler
	user *user.Handler
	pr   *pullrequest.Handler
}

func NewServerHandler(teamSvc service.TeamService, userSvc service.UserService, prSvc service.PullRequestService) *ServerHandler {
	t := team.New(teamSvc)
	u := user.New(userSvc, prSvc)
	p := pullrequest.New(prSvc)
	return &ServerHandler{team: t, user: u, pr: p}
}

func (h *ServerHandler) RegisterRoutes(router *chi.Mux) {
	wrapper := &api.ServerInterfaceWrapper{
		Handler: h,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
	}

	router.Route("/v1", func(r chi.Router) {
		router.Route("/team", func(r chi.Router) {
			r.Post("/add", wrapper.PostTeamAdd)
			r.Get("/get", wrapper.GetTeamGet)
		})

		router.Route("/users", func(r chi.Router) {
			r.Post("/setIsActive", wrapper.PostUsersSetIsActive)
			r.Get("/getReview", wrapper.GetUsersGetReview)
			r.Post("/deactivateBatch", wrapper.PostUsersDeactivateBatch)
		})

		router.Route("/pullRequest", func(r chi.Router) {
			r.Post("/create", wrapper.PostPullRequestCreate)
			r.Post("/merge", wrapper.PostPullRequestMerge)
			r.Post("/reassign", wrapper.PostPullRequestReassign)
		})

		router.Get("/stats", h.pr.GetStats)
	})
}

func (h *ServerHandler) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	h.team.PostTeamAdd(w, r)
}

func (h *ServerHandler) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	h.team.GetTeamGet(w, r, params)
}

func (h *ServerHandler) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	h.user.PostUsersSetIsActive(w, r)
}

func (h *ServerHandler) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	h.pr.PostPullRequestCreate(w, r)
}

func (h *ServerHandler) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	h.pr.PostPullRequestMerge(w, r)
}

func (h *ServerHandler) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	h.pr.PostPullRequestReassign(w, r)
}

func (h *ServerHandler) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	h.user.GetUsersGetReview(w, r, params)
}

func (h *ServerHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	h.pr.GetStats(w, r)
}

func (h *ServerHandler) PostUsersDeactivateBatch(w http.ResponseWriter, r *http.Request) {
	h.user.PostUsersDeactivateBatch(w, r)
}
