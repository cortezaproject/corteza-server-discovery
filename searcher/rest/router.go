package rest

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-server-discovery/searcher/rest/handlers"
	"github.com/cortezaproject/corteza-server/pkg/errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	"net/http"
)

func MountRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(HttpTokenValidator())
			handlers.NewSearch(Search()).MountRoutes(r)
		})
	}
}

// HttpTokenValidator checks if there is a token with identity
func HttpTokenValidator() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := verifyToken(r.Context())
			if err != nil && !errors.Is(err, jwtauth.ErrNoTokenFound) {
				errors.ProperlyServeHTTP(w, r, err, false)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// pulls token from context and validates access-token
func verifyToken(ctx context.Context) (err error) {
	var token *jwt.Token
	if token, _, err = jwtauth.FromContext(ctx); err != nil {
		return
	}

	if token == nil {
		return fmt.Errorf("unauthorized")
	}

	return
}
