package main

import (
	"github.com/cortezaproject/corteza-discovery-indexer/app"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7/esutil"
)

var _ *spew.ConfigState = nil
var _ esutil.BulkIndexer

func main() {
	ctx := cli.Context()

	a, err := app.New()
	cli.HandleError(err)

	err = a.InitService(ctx)
	cli.HandleError(err)
}
