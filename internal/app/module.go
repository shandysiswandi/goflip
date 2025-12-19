package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/shandysiswandi/goflip/internal/flip"
)

func (a *App) initModules() {
	if a.config.GetBool("modules.flip.enabled") {
		closer, err := flip.New(flip.Dependency{
			Config:    a.config,
			Router:    a.router,
			Goroutine: a.goroutine,
			Context:   a.ctx,
			ID:        a.uuid,
		})
		if err != nil {
			slog.Error("failed to init module flip", "error", err)
			os.Exit(1)
		}
		if closer != nil {
			if a.closerFn == nil {
				a.closerFn = map[string]func(context.Context) error{}
			}
			a.closerFn["Flip"] = closer
		}
	}
}
