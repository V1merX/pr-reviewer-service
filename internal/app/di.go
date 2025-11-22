package app

import (
	"log/slog"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/V1merX/pr-reviewer-service/internal/config"
	httpserver "github.com/V1merX/pr-reviewer-service/internal/http"
	"github.com/V1merX/pr-reviewer-service/internal/repository"
	pgrepo "github.com/V1merX/pr-reviewer-service/internal/repository/postgres"
	"github.com/V1merX/pr-reviewer-service/internal/service"
	pullrequestService "github.com/V1merX/pr-reviewer-service/internal/service/pullrequest"
	teamService "github.com/V1merX/pr-reviewer-service/internal/service/team"
	userService "github.com/V1merX/pr-reviewer-service/internal/service/user"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type diContainer struct {
	cfg    *config.Config
	db     *sqlx.DB
	logger *slog.Logger

	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
	prRepo   repository.PullRequestRepository

	teamService service.TeamService
	userService service.UserService
	prService   service.PullRequestService

	httpServer *httpserver.Server
	cfgPath    string
}

func NewDIContainer(cfgPath string) *diContainer {
	return &diContainer{cfgPath: cfgPath}
}

func (d *diContainer) Logger(env string) *slog.Logger {
	if d.logger != nil {
		return d.logger
	}

	switch env {
	case envLocal:
		d.logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		d.logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		d.logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		d.logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return d.logger
}

func (d *diContainer) Config() (*config.Config, error) {
	if d.cfg == nil {
		cfg, err := config.Load(d.cfgPath)
		if err != nil {
			return nil, err
		}
		d.cfg = cfg
	}
	return d.cfg, nil
}

func (d *diContainer) DB() (*sqlx.DB, error) {
	if d.db == nil {
		cfg, err := d.Config()
		if err != nil {
			return nil, err
		}
		db, err := pgrepo.Open(cfg)
		if err != nil {
			return nil, err
		}
		d.db = db
	}
	return d.db, nil
}

func (d *diContainer) TeamRepository() (repository.TeamRepository, error) {
	if d.teamRepo == nil {
		db, err := d.DB()
		if err != nil {
			return nil, err
		}
		d.teamRepo = pgrepo.NewTeamRepository(db, d.Logger(d.cfg.Server.Env))
	}
	return d.teamRepo, nil
}

func (d *diContainer) UserRepository() (repository.UserRepository, error) {
	if d.userRepo == nil {
		db, err := d.DB()
		if err != nil {
			return nil, err
		}
		d.userRepo = pgrepo.NewUserRepository(db, d.Logger(d.cfg.Server.Env))
	}
	return d.userRepo, nil
}

func (d *diContainer) PullRequestRepository() (repository.PullRequestRepository, error) {
	if d.prRepo == nil {
		db, err := d.DB()
		if err != nil {
			return nil, err
		}
		d.prRepo = pgrepo.NewPullRequestRepository(db, d.Logger(d.cfg.Server.Env))
	}
	return d.prRepo, nil
}

func (d *diContainer) TeamService() (service.TeamService, error) {
	if d.teamService == nil {
		repo, err := d.TeamRepository()
		if err != nil {
			return nil, err
		}
		d.teamService = teamService.NewService(repo, d.Logger(d.cfg.Server.Env))
	}
	return d.teamService, nil
}

func (d *diContainer) UserService() (service.UserService, error) {
	if d.userService == nil {
		repo, err := d.UserRepository()
		if err != nil {
			return nil, err
		}
		d.userService = userService.NewService(d.Logger(d.cfg.Server.Env), repo)
	}
	return d.userService, nil
}

func (d *diContainer) PullRequestService() (service.PullRequestService, error) {
	if d.prService == nil {
		prRepo, err := d.PullRequestRepository()
		if err != nil {
			return nil, err
		}
		teamRepo, err := d.TeamRepository()
		if err != nil {
			return nil, err
		}
		userRepo, err := d.UserRepository()
		if err != nil {
			return nil, err
		}
		d.prService = pullrequestService.NewService(d.Logger(d.cfg.Server.Env), prRepo, teamRepo, userRepo)
	}
	return d.prService, nil
}

func (d *diContainer) HTTPServer() (*httpserver.Server, error) {
	if d.httpServer == nil {
		cfg, err := d.Config()
		if err != nil {
			return nil, err
		}
		logger := d.Logger(cfg.Server.Env)
		teamSvc, err := d.TeamService()
		if err != nil {
			return nil, err
		}
		userSvc, err := d.UserService()
		if err != nil {
			return nil, err
		}
		prSvc, err := d.PullRequestService()
		if err != nil {
			return nil, err
		}
		d.httpServer = httpserver.New(cfg, logger, teamSvc, userSvc, prSvc)
	}
	return d.httpServer, nil
}
