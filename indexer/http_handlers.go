package indexer

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
)

var _ = spew.Dump

type (
	handlers struct {
		log *zap.Logger
		esc *elasticsearch.Client
		api *apiClient
	}
)

func Handlers(r chi.Router, log *zap.Logger, esc *elasticsearch.Client, api *apiClient) *handlers {
	h := &handlers{
		esc: esc,
		log: log,
		api: api,
	}

	r.Use()

	r.Get("/healthcheck", h.Healthcheck)

	return h
}

func (h handlers) Healthcheck(w http.ResponseWriter, r *http.Request) {
	res, err := h.esc.Ping(
		h.esc.Ping.WithContext(r.Context()),
	)

	if validElasticResponse(h.log, res, err) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "unhealthy")
		return
	}

	defer res.Body.Close()

	_, _ = fmt.Fprintf(w, "healthy")
}
