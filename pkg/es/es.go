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
)

type (
	es struct {
		log *zap.Logger
		opt options.EsOpt
		//esc *elasticsearch.Client
		//esb esutil.BulkIndexer
	}

	Service interface {
		EsClient() *elasticsearch.Client
		EsBulk() esutil.BulkIndexer
		Watch(ctx context.Context)
	}

	//apiClientService interface {
	//	HttpClient() *http.Client
	//	Mappings() (*http.Request, error)
	//	Resources(string, url.Values) (*http.Request, error)
	//	Request(string) (*http.Request, error)
	//	Authenticate() error
	//}
)

func ES(log *zap.Logger, opt options.EsOpt) (out *es, err error) {
	out = &es{log: log, opt: opt}

	//out.esc, err = out.newClient()
	//if err != nil {
	//	return
	//}

	//out.esb, err = out.newBulkIndexer()
	//if err != nil {
	//	return
	//}

	return
}

func (es *es) Client() (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses:            es.opt.Addresses,
		EnableRetryOnTimeout: es.opt.EnableRetryOnTimeout,
		MaxRetries:           es.opt.MaxRetries,
	})
}

func (es *es) BulkIndexer() (esutil.BulkIndexer, error) {
	client, err := es.Client()
	if err != nil {
		return nil, err
	}

	return esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     client,
		FlushBytes: 5e+5,
	})
}

//func (es *es) EsClient() *elasticsearch.Client {
//	return es.esc
//}
//
//func (es *es) EsBulk() esutil.BulkIndexer {
//	return es.esb
//}

//func (es *es) Watch(ctx context.Context) {
//	fmt.Println("ticker: ", es.opt.IndexInterval)
//	//if es.opt.IndexInterval > 0 {
//		ticker := time.NewTicker(time.Second * 1)
//		go func() {
//			defer ticker.Stop()
//
//			for {
//				select {
//				case <-ctx.Done():
//					fmt.Println("Tickinggg overrrr ", time.Now())
//
//					return
//				case <-ticker.C:
//					fmt.Println("Tickinggg ", time.Now())
//					err := DefaultMapper.Mappings(ctx, "private")
//					if err != nil {
//						es.log.Error("failed to mapping", zap.Error(err))
//					}
//
//					err = DefaultReIndexer.ReindexAll(ctx, "private")
//					if err != nil {
//						es.log.Error("failed to reindex", zap.Error(err))
//					}
//
//					esb, err := DefaultEs.EsBulk()
//					if err != nil {
//						es.log.Error("failed to start bulk indexer", zap.Error(err))
//					}
//
//					if err := esb.Close(ctx); err != nil {
//						es.log.Error("failed to close bulk indexer", zap.Error(err))
//					}
//				}
//			}
//		}()
//
//		es.log.Debug("watcher initialized")
//	//}
//}

//func (es *es) Watch(ctx context.Context) {
//	ticker := time.NewTicker(1 * time.Second * 15)
//	for _ = range ticker.C {
//		err := indexer.DefaultMapper.Mappings(ctx, "private")
//		if err != nil {
//			es.log.Error("failed to mapping", zap.Error(err))
//		}
//
//		err = indexer.DefaultReIndexer.ReindexAll(ctx, "private")
//		if err != nil {
//			es.log.Error("failed to reindex", zap.Error(err))
//		}
//	}
//}

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
