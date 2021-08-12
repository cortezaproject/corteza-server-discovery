package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-indexer/pkg/options"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

type (
	es struct {
		log *zap.Logger
		opt options.EsOpt
	}

	esService interface {
		EsClient() (*elasticsearch.Client, error)
		EsBulk() (esutil.BulkIndexer, error)
		Watch(ctx context.Context)
	}

	apiClientService interface {
		HttpClient() *http.Client
		Mappings() (*http.Request, error)
		Resources(string, url.Values) (*http.Request, error)
		Request(string) (*http.Request, error)
		Authenticate() error
	}
)

func ES(log *zap.Logger, opt options.EsOpt) *es {
	return &es{log, opt}
}

func (es *es) EsClient() (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses:            es.opt.Addresses,
		EnableRetryOnTimeout: es.opt.EnableRetryOnTimeout,
		MaxRetries:           es.opt.MaxRetries,
	})
}

func (es *es) EsBulk() (esutil.BulkIndexer, error) {
	esc, err := es.EsClient()
	if err != nil {
		return nil, err
	}
	return esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     esc,
		FlushBytes: 5e+5,
	})
}

func (es *es) Watch(ctx context.Context) {
	fmt.Println("ticker: ", es.opt.IndexInterval)
	if es.opt.IndexInterval > 0 {
		ticker := time.NewTicker(time.Second * time.Duration(es.opt.IndexInterval))
		go func() {
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					err := DefaultMapper.Mappings(ctx, "private")
					if err != nil {
						es.log.Error("failed to mapping", zap.Error(err))
					}

					err = DefaultReIndexer.ReindexAll(ctx, "private")
					if err != nil {
						es.log.Error("failed to reindex", zap.Error(err))
					}

					esb, err := DefaultEs.EsBulk()
					if err != nil {
						es.log.Error("failed to start bulk indexer", zap.Error(err))
					}

					if err := esb.Close(ctx); err != nil {
						es.log.Error("failed to close bulk indexer", zap.Error(err))
					}
				}
			}
		}()

		es.log.Debug("watcher initialized")
	}
}

func ValidElasticResponse(log *zap.Logger, res *esapi.Response, err error) error {
	if err != nil {
		return fmt.Errorf("failed to get response from search backend: %w", err)
	}

	if res.IsError() {
		defer res.Body.Close()
		var rsp struct {
			Error struct {
				Type   string
				Reason string
			}
		}

		if err := json.NewDecoder(res.Body).Decode(&rsp); err != nil {
			return fmt.Errorf("could not parse response body: %w", err)
		} else {
			return fmt.Errorf("search backend responded with an error: %s (type: %s, status: %s)", rsp.Error.Reason, rsp.Error.Type, res.Status())
		}
	}

	return nil
}
