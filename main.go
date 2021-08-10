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

	//{
	//	wg := &sync.WaitGroup{}
	//
	//	{
	//		a.HttpServer = server.New(a.Log, a.Opt.Environment, a.Opt.HTTPServer, a.Opt.WaitFor, a.Opt.Searcher)
	//
	//		wg.Add(1)
	//		go func() {
	//			a.HttpServer.Serve(ctx)
	//			wg.Done()
	//		}()
	//	}
	//
	//	err = a.Activate(ctx)
	//	cli.HandleError(err)
	//
	//	a.HttpServer.Activate(a.MountHttpRoutes)
	//
	//	// Wait for all servers to be done
	//	wg.Wait()
	//
	//	a.HttpServer.Shutdown()
	//}

	err = a.Activate(ctx)
	cli.HandleError(err)
}
