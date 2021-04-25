package main

import (
	"github.com/cortezaproject/corteza-discovery-indexer/indexer"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.uber.org/zap"
)

var _ *spew.ConfigState = nil
var _ esutil.BulkIndexer

func main() {
	log := logger.MakeDebugLogger().WithOptions(zap.AddStacktrace(zap.PanicLevel))
	ctx := cli.Context()
	cfg, err := getConfig()
	cli.HandleError(err)

	api, err := indexer.ApiClient(cfg.cortezaDiscoveryAPI, cfg.cortezaAuth, cfg.schemas[0].clientKey, cfg.schemas[0].clientSecret)
	cli.HandleError(err)

	//
	esc, err := indexer.EsClient(cfg.es.addresses)
	cli.HandleError(err)

	cli.HandleError(indexer.Mappings(ctx, log, esc, api, "private"))

	esb, err := indexer.EsBulk(esc)
	cli.HandleError(err)

	_ = esb
	cli.HandleError(indexer.ReindexAll(ctx, log, esb, api, "private"))

	if err := esb.Close(ctx); err != nil {
		log.Error("failed to close bulk indexer", zap.Error(err))
	}
}
