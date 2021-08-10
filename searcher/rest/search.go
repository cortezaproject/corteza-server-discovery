package rest

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/searcher/rest/request"
)

type (
	search struct {
	}
)

func Search() *search {
	return &search{}
}

func (s search) SearchResources(ctx context.Context, resources *request.SearchResources) (interface{}, error) {
	fmt.Println("At SearchResources: ")
	return nil, nil
	//panic("implement me")
}

func (s search) Sandbox(ctx context.Context, sandbox *request.SearchSandbox) {
	panic("implement me")
}

func (s search) HealthCheck(ctx context.Context, check *request.SearchHealthCheck) (interface{}, error) {
	panic("implement me")
}
