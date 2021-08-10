package server

import (
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"strings"
)

// routes used when server is in waiting mode
func waitingRoutes(log *zap.Logger, httpOpt options.HttpServerOpt) (r chi.Router) {
	r = chi.NewRouter()
	r.Use(handleCORS)

	return
}

// routes used when server in shutdown mode
func shutdownRoutes() (r chi.Router) {
	r = chi.NewRouter()

	return
}

// routes used when in active mode
func activeRoutes(log *zap.Logger, mountable []func(r chi.Router), envOpt options.EnvironmentOpt, httpOpt options.HttpServerOpt, searcherOpt options.SearcherOpt) (r chi.Router) {
	r = chi.NewRouter()
	r.Use(handleCORS)

	fmt.Println(">>>>>>>>: ", "/"+strings.TrimPrefix(httpOpt.BaseUrl, "/"))
	r.Route("/"+strings.TrimPrefix(httpOpt.BaseUrl, "/"), func(r chi.Router) {
		//fmt.Println("httpOpt.BaseUrl: ", httpOpt.BaseUrl)
		//r.Route(httpOpt.BaseUrl, func(r chi.Router) {
		// Handle panic (sets 500 server error headers)
		//r.Use(handlePanic)

		// Base middleware, CORS, RealIP, RequestID, context-logger
		r.Use(BaseMiddleware(envOpt.IsProduction(), log)...)

		// Verifies JWT in headers, cookies, ... @todo
		//r.Use(auth.HttpTokenVerifier)
		//if len(searcherOpt.JwtSecret) == 0 {
		//	log.Warn(fmt.Sprintf("JWT secret not set, access to private indexes disabled"))
		//} else {
		//	r.Use(jwtauth.Verifier(jwtauth.New(jwt.SigningMethodHS512.Alg(), searcherOpt.JwtSecret, nil)))
		//}

		for _, mount := range mountable {
			mount(r)
		}

	})

	//if httpOpt.BaseUrl != "/" {
	//	r.Handle("/", http.RedirectHandler(httpOpt.BaseUrl, http.StatusTemporaryRedirect))
	//}

	return
}
