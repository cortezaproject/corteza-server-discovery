package app

import (
	"context"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/es"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type (
	httpApiServer interface {
		MountRoutes(mm ...func(chi.Router))
		Serve(ctx context.Context)
	}

	IndexerApp struct {
		Log *zap.Logger
		Opt *options.Options

		// Servers
		HttpServer httpApiServer
	}
)

var (
	_ *spew.ConfigState = nil
	_ esutil.BulkIndexer
)

func New() (app *IndexerApp, err error) {
	app = &IndexerApp{
		Log: logger.MakeDebugLogger().WithOptions(zap.AddStacktrace(zap.PanicLevel)),
	}
	app.Opt, err = options.Init()
	if err != nil {
		return
	}

	return
}

func (app IndexerApp) Serve(ctx context.Context) (err error) {
	return
}

func (app IndexerApp) InitService(ctx context.Context) (err error) {
	// Initialize indexer service
	err = es.Initialize(ctx, app.Log, es.Config{
		ES:      app.Opt.ES,
		Indexer: app.Opt.Indexer,
	})
	if err != nil {
		return err
	}

	return
}
