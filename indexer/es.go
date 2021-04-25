package indexer

import (
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
)

func EsClient(aa []string) (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses:            aa,
		EnableRetryOnTimeout: true,
		MaxRetries:           5,
	})
}

func EsBulk(esc *elasticsearch.Client) (esutil.BulkIndexer, error) {
	return esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     esc,
		FlushBytes: 5e+5,
	})
}
