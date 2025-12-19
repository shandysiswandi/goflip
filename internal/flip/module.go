package flip

import (
	"context"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/event"
	"github.com/shandysiswandi/goflip/internal/flip/inbound"
	"github.com/shandysiswandi/goflip/internal/flip/store"
	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgroutine"
	"github.com/shandysiswandi/goflip/internal/pkg/pkguid"
)

type Dependency struct {
	Config    pkgconfig.Config
	Goroutine *pkgroutine.Manager
	Router    *pkgrouter.Router
	Context   context.Context
	ID        pkguid.StringID
}

func New(dep Dependency) (func(context.Context) error, error) {
	storage := store.NewInMemoryStore()
	bus := event.NewBus(512)
	consumer := event.NewReconciliationConsumer(bus, event.NoopReconciler{}, event.ConsumerConfig{
		Workers:     4,
		MaxRetries:  3,
		BaseBackoff: 200 * time.Millisecond,
	})
	consumer.Start()

	if dep.ID == nil {
		dep.ID = pkguid.NewUUID()
	}

	uc := usecase.New(usecase.Dependency{
		Store:   storage,
		Events:  bus,
		Runner:  dep.Goroutine,
		Clock:   nil,
		ID:      dep.ID,
		RootCtx: dep.Context,
	})

	inbound.RegisterHTTPEndpoint(dep.Router, uc)

	return consumer.Stop, nil
}
