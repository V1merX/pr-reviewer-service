package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/V1merX/pr-reviewer-service/internal/config"
	"github.com/V1merX/pr-reviewer-service/internal/http/handler"
	"github.com/V1merX/pr-reviewer-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Config  *config.Config
	Router  *chi.Mux
	Logger  *slog.Logger
	Handler *handler.ServerHandler
}

func New(config *config.Config, logger *slog.Logger, teamService service.TeamService, userService service.UserService, prService service.PullRequestService) *Server {
	return &Server{
		Config:  config,
		Router:  chi.NewRouter(),
		Logger:  logger,
		Handler: handler.NewServerHandler(teamService, userService, prService),
	}
}

func (s *Server) Run() error {
	s.configureRouter()
	srv := &http.Server{
		Addr:         s.Config.Server.Port,
		Handler:      s.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) configureRouter() {
	s.Router.Use(middleware.DefaultLogger)
	s.Handler.RegisterRoutes(s.Router)
}
