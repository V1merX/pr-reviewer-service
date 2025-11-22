package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	httpserver "github.com/V1merX/pr-reviewer-service/internal/http"
	pgrepo "github.com/V1merX/pr-reviewer-service/internal/repository/postgres"
	"github.com/jmoiron/sqlx"
)

type App struct {
	diContainer *diContainer
	HttpServer  *httpserver.Server
	db          *sqlx.DB
}

func New() (*App, error) {
	a := &App{}

	if err := a.initDeps(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) initDeps() error {
	steps := []func() error{
		a.initDI,
		a.initHTTPServer,
	}

	for _, s := range steps {
		if err := s(); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initDI() error {
	a.diContainer = NewDIContainer("./config")

	db, err := a.diContainer.DB()
	if err != nil {
		return err
	}
	a.db = db
	return nil
}

func (a *App) initHTTPServer() error {
	srv, err := a.diContainer.HTTPServer()
	if err != nil {
		return err
	}
	a.HttpServer = srv
	return nil
}

func (a *App) Run(ctx context.Context) error {
	go func() {
		if err := a.runHTTPServer(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down server...")

	if a.db != nil {
		pgrepo.Close(a.db)
	}

	log.Printf("Server stopped")
	return nil
}

func (a *App) runHTTPServer() error {
	if a.HttpServer == nil {
		return nil
	}
	return a.HttpServer.Run()
}
