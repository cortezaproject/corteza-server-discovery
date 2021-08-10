package rest

import (
	"github.com/cortezaproject/corteza-discovery-indexer/searcher/rest/handlers"
	"github.com/go-chi/chi/v5"
)

func MountRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			//r.Use(auth.AccessTokenCheck("discovery"))

			handlers.NewSearch(Search())
		})
	}
}
