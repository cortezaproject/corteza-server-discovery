package es

import (
	"context"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/api"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"go.uber.org/zap"
)

type (
	Config struct {
		ES      options.EsOpt
		Indexer options.IndexerOpt
	}

	//esService interface {
	//	EsClient() (*elasticsearch.Client, error)
	//	EsBulk() (esutil.BulkIndexer, error)
	//	Watch(ctx context.Context)
	//}

	//apiClientService interface {
	//	HttpClient() *http.Client
	//	Mappings() (*http.Request, error)
	//	Resources(string, url.Values) (*http.Request, error)
	//	Request(string) (*http.Request, error)
	//	Authenticate() error
	//}

	mappingService interface {
		Mappings(ctx context.Context, indexPrefix string) (err error)
		ConfigurationMapping(ctx context.Context) (err error)
	}

	reIndexService interface {
		ReindexAll(ctx context.Context, indexPrefix string) error
	}
)

var (
	DefaultEs        esService
	DefaultApiClient apiClientService
	DefaultMapper    mappingService
	DefaultReIndexer reIndexService
)

func Initialize(ctx context.Context, log *zap.Logger, c Config) (err error) {
	DefaultEs, err = ES(log, c.ES)
	if err != nil {
		return
	}

	DefaultApiClient, err = api.Client(c.Indexer)
	if err != nil {
		return
	}

	DefaultMapper = Mapper(log, DefaultEs, DefaultApiClient)

	err = DefaultMapper.ConfigurationMapping(ctx)
	if err != nil {
		return err
	}
	// @todo: 2.0 private/public/protected indexing
	err = DefaultMapper.Mappings(ctx, "private")
	if err != nil {
		return err
	}

	DefaultReIndexer = ReIndexer(log, DefaultEs, DefaultApiClient)
	err = DefaultReIndexer.ReindexAll(ctx, "private")
	if err != nil {
		return err
	}

	// Initiate watchers
	Watchers(ctx)

	return
}

func Watchers(ctx context.Context) {
	// Initiate watcher for reindexing resource
	DefaultEs.Watch(ctx)
	return
}
