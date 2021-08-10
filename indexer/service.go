package indexer

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/api"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/es"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/es/mapping"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/es/reindex"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.uber.org/zap"
	"net/http"
	"net/url"
)

type (
	Config struct {
		Corteza options.CortezaOpt
		ES      options.EsOpt
		Indexer options.IndexerOpt
	}

	esService interface {
		Client() (*elasticsearch.Client, error)
		BulkIndexer() (esutil.BulkIndexer, error)
	}

	apiClientService interface {
		HttpClient() *http.Client
		Mappings() (*http.Request, error)
		Resources(string, url.Values) (*http.Request, error)
		Request(string) (*http.Request, error)
		Authenticate() error
	}

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
	DefaultEs, err = es.ES(log, c.ES)
	if err != nil {
		return
	}

	schema := c.Indexer.Schemas[0]
	if len(schema.ClientKey) == 0 || len(schema.ClientSecret) == 0 {
		return fmt.Errorf("client key and secret is missing")
	}
	DefaultApiClient, err = api.Client(c.Corteza, schema.ClientKey, schema.ClientSecret)
	if err != nil {
		return
	}

	DefaultMapper = mapping.Mapper(log, DefaultEs, DefaultApiClient)

	err = DefaultMapper.ConfigurationMapping(ctx)
	if err != nil {
		return err
	}

	// @todo: private/public/protected indexing
	err = DefaultMapper.Mappings(ctx, "private")
	if err != nil {
		return err
	}

	DefaultReIndexer = reindex.ReIndexer(log, DefaultEs, DefaultApiClient)
	err = DefaultReIndexer.ReindexAll(ctx, "private")
	if err != nil {
		return err
	}

	return
}

func Watchers(ctx context.Context) {
	// Initiate watcher for reindexing resource
	//DefaultMapper.Watch(ctx)
	//DefaultReIndexer.Watch(ctx)

	return
}
