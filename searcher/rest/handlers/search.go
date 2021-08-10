package handlers

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/searcher/rest/request"
	"github.com/cortezaproject/corteza-server/pkg/api"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type (
	// SearchAPI Internal API interface
	SearchAPI interface {
		SearchResources(context.Context, *request.SearchResources) (interface{}, error)
		Sandbox(context.Context, *request.SearchSandbox)
		HealthCheck(context.Context, *request.SearchHealthCheck) (interface{}, error)
	}

	// Search HTTP API interface
	Search struct {
		SearchResources func(http.ResponseWriter, *http.Request)
		Sandbox         func(http.ResponseWriter, *http.Request)
		HealthCheck     func(http.ResponseWriter, *http.Request)
	}
)

func NewSearch(h SearchAPI) *Search {
	return &Search{
		SearchResources: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			params := request.NewSearchListResources()
			if err := params.Fill(r); err != nil {
				api.Send(w, r, err)
				return
			}

			value, err := h.SearchResources(r.Context(), params)
			if err != nil {
				api.Send(w, r, err)
				return
			}

			api.Send(w, r, value)
		},
		Sandbox: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			params := request.NewSearchSandbox()
			if err := params.Fill(r); err != nil {
				api.Send(w, r, err)
				return
			}

			fmt.Println("I am here at least")
			//value, err := h.Sandbox(r.Context(), params)
			//if err != nil {
			//	api.Send(w, r, err)
			//	return
			//}

			p := "." + r.URL.Path
			if p == "./" {
				p = "./sandbox/index.html"
			}
			http.ServeFile(w, r, p)

			//api.Send(w, r, value)
		},
		HealthCheck: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			params := request.NewSearchHealthCheck()
			if err := params.Fill(r); err != nil {
				api.Send(w, r, err)
				return
			}

			value, err := h.HealthCheck(r.Context(), params)
			if err != nil {
				api.Send(w, r, err)
				return
			}

			api.Send(w, r, value)
		},
	}
}

func (h Search) MountRoutes(r chi.Router, middlewares ...func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares...)
		r.Get("/two", h.SearchResources)
		r.Get("/sandbox", h.Sandbox)
		// @todo refactor this
		r.Get("/healthcheck", h.HealthCheck)
	})
}
