package searcher

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/api"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/es"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/dgrijalva/jwt-go"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth"
	"go.uber.org/zap"
	"net"
	"net/http"
)

type (
	Config struct {
		Corteza    options.CortezaOpt
		ES         options.EsOpt
		HttpServer options.HttpServerOpt
		Searcher   options.SearcherOpt
	}

	esService interface {
		Client() (*elasticsearch.Client, error)
		BulkIndexer() (esutil.BulkIndexer, error)
	}

	apiClientService interface {
		HttpClient() *http.Client
		Namespaces() (*http.Request, error)
		Modules(uint64) (*http.Request, error)
		Request(string) (*http.Request, error)
		Authenticate() error
	}
)

var (
	DefaultEs        esService
	DefaultApiClient apiClientService
)

func Initialize(ctx context.Context, log *zap.Logger, c Config) (err error) {
	DefaultEs, err = es.ES(log, c.ES)
	if err != nil {
		return
	}

	DefaultApiClient, err = api.Client(c.Corteza, c.Searcher.ClientKey, c.Searcher.ClientSecret)
	if err != nil {
		return
	}

	esc, err := DefaultEs.Client()
	cli.HandleError(err)

	StartHttpServer(ctx, log, c.HttpServer.Addr, func() http.Handler {
		router := chi.NewRouter()
		router.Use(handleCORS)
		router.Use(middleware.StripSlashes)
		router.Use(middleware.RealIP)
		router.Use(middleware.RequestID)

		if len(c.Searcher.JwtSecret) == 0 {
			log.Warn(fmt.Sprintf("JWT secret not set, access to private indexes disabled"))
		} else {
			router.Use(jwtauth.Verifier(jwtauth.New(jwt.SigningMethodHS512.Alg(), c.Searcher.JwtSecret, nil)))
		}

		// @todo If we want to prevent any kind of anonymous access
		//router.Use(jwtauth.Authenticator)

		Handlers(router, log, esc, DefaultApiClient)

		return router
	}())

	return
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

// Sets up default CORS rules to use as a middleware
func handleCORS(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-ID"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}).Handler(next)
}
