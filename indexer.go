package main

import (
	"context"
	"github.com/cortezaproject/corteza-discovery-indexer/indexer"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"net"
	"net/http"
)

var _ *spew.ConfigState = nil
var _ esutil.BulkIndexer

func main() {
	log := logger.MakeDebugLogger().WithOptions(zap.AddStacktrace(zap.PanicLevel))
	ctx := cli.Context()
	cfg, err := getConfig()
	cli.HandleError(err)

	api, err := indexer.ApiClient(cfg.cortezaDiscoveryAPI, cfg.cortezaAuth, cfg.schemas[0].clientKey, cfg.schemas[0].clientSecret)
	cli.HandleError(err)

	esc, err := indexer.EsClient(cfg.es.addresses)
	cli.HandleError(err)

	cli.HandleError(indexer.Mappings(ctx, log, esc, api, "private"))

	esb, err := indexer.EsBulk(esc)
	cli.HandleError(err)

	_ = esb
	cli.HandleError(indexer.ReindexAll(ctx, log, esb, api, "private"))

	if err := esb.Close(ctx); err != nil {
		log.Error("failed to close bulk indexer", zap.Error(err))
	}

	StartHttpServer(ctx, log, cfg.httpAddr, func() http.Handler {
		router := chi.NewRouter()
		router.Use(middleware.StripSlashes)
		router.Use(middleware.RealIP)
		router.Use(middleware.RequestID)

		// @todo If we want to prevent any kind of anonymous access
		//router.Use(jwtauth.Authenticator)

		indexer.Handlers(router, log, esc, api)

		return router
	}())
}

func StartHttpServer(ctx context.Context, log *zap.Logger, addr string, h http.Handler) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("cannot start server", zap.Error(err))
		return
	}

	go func() {
		srv := http.Server{
			Handler: h,
			BaseContext: func(listener net.Listener) context.Context {
				return ctx
			},
		}
		log.Info("http server started", zap.String("addr", addr))
		err = srv.Serve(listener)
	}()
	<-ctx.Done()
}
