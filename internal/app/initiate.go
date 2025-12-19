package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgroutine"
	"github.com/shandysiswandi/goflip/internal/pkg/pkguid"
)

func (a *App) initConfig() {
	path := "/config/config.yaml"
	if os.Getenv("LOCAL") == "true" {
		path = "./config/config.yaml"
	}

	cfg, err := pkgconfig.NewViper(path)
	if err != nil {
		slog.Error("failed to init config", "error", err)
		os.Exit(1)
	}

	//nolint:errcheck,gosec // ignore error
	os.Setenv("TZ", cfg.GetString("tz"))

	a.config = cfg
}

func (a *App) initLibraries() {
	a.goroutine = pkgroutine.NewManager(100)
	a.uuid = pkguid.NewUUID()
}

func (a *App) initHTTPServer() {
	a.router = pkgrouter.NewRouter(a.uuid)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	a.httpServer = &http.Server{
		Addr:              a.config.GetString("server.address.http"),
		Handler:           corsHandler.Handler(a.router),
		ReadHeaderTimeout: 10 * time.Second,
	}
}

//nolint:unparam // is always nil
func (a *App) initClosers() {
	if a.closerFn == nil {
		a.closerFn = map[string]func(context.Context) error{}
	}

	a.closerFn["HTTP Server"] = func(ctx context.Context) error {
		return a.httpServer.Shutdown(ctx)
	}
	a.closerFn["Config"] = func(context.Context) error {
		return a.config.Close()
	}
}
