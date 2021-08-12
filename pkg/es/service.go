package es

import (
	"context"
	"fmt"
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
	DefaultEs = ES(log, c.ES)
	if err != nil {
		return
	}

	DefaultApiClient, err = api.Client(
		c.Indexer.CortezaDiscoveryAPI,
		c.Indexer.CortezaAuth,
		c.Indexer.Schemas[0].ClientKey,
		c.Indexer.Schemas[0].ClientSecret,
	)
	if err != nil {
		return
	}

	DefaultMapper = Mapper(log, DefaultEs, DefaultApiClient)

	// @todo: private/public indexing
	err = DefaultMapper.Mappings(ctx, "private")
	if err != nil {
		return err
	}

	DefaultReIndexer = ReIndexer(log, DefaultEs, DefaultApiClient)
	err = DefaultReIndexer.ReindexAll(ctx, "private")
	if err != nil {
		return err
	}

	esb, err := DefaultEs.EsBulk()
	if err != nil {
		return err
	}

	if err := esb.Close(ctx); err != nil {
		return fmt.Errorf("failed to close bulk indexer: %w", err)
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
