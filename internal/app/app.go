package app

import (
	"context"
	"net/http"

	"github.com/shandysiswandi/goflip/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/goflip/internal/pkg/pkglog"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgroutine"
	"github.com/shandysiswandi/goflip/internal/pkg/pkguid"
)

type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	// configuration
	config pkgconfig.Config

	// libraries
	uuid      pkguid.StringID
	goroutine *pkgroutine.Manager

	// resources

	// server
	router     *pkgrouter.Router
	httpServer *http.Server

	//
	closerFn map[string]func(context.Context) error
}

func New() *App {
	pkglog.InitLogging()

	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		ctx:    ctx,
		cancel: cancel,
	}

	app.initConfig()
	app.initLibraries()
	app.initHTTPServer()
	app.initModules()
	app.initClosers()

	return app
}
