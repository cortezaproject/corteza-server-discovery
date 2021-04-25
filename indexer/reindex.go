package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

type (
	docsSources struct {
		endpoint string
		index    string
		callback func(*document)
	}

	rspDiscoveryDocuments struct {
		Error *struct {
			Message string
		}
		Response *struct {
			Filter struct {
				NextPage string
			}

			Documents []*document
		}
	}

	// auxiliary struct for parsing indexable documents from Discovery API
	document struct {
		ID     string
		Index  string
		Source json.RawMessage
	}
)

const (
	indexTpl = "corteza-%s-%s"
)

func ReindexAll(ctx context.Context, log *zap.Logger, esb esutil.BulkIndexer, api *apiClient, indexPrefix string) error {
	var (
		srcQueue = make(chan *docsSources, 100)
		bErr     = reindexManager(ctx, log, esb, api, indexPrefix, srcQueue)
	)

	srcQueue <- &docsSources{
		endpoint: "/system/users",
		index:    "system-users",
	}

	postProcModules := func(namespaceID string) func(d *document) {
		return func(d *document) {
			srcQueue <- &docsSources{
				endpoint: fmt.Sprintf("/compose/namespaces/%s/modules/%s/records", namespaceID, d.ID),
				index:    fmt.Sprintf("compose-records-%s-%s", namespaceID, d.ID),
			}
		}
	}

	postProcNamespaces := func(d *document) {
		srcQueue <- &docsSources{
			endpoint: fmt.Sprintf("/compose/namespaces/%s/modules", d.ID),
			index:    "compose-modules",
			callback: postProcModules(d.ID),
		}
	}

	_ = postProcModules
	_ = postProcNamespaces

	srcQueue <- &docsSources{
		endpoint: "/compose/namespaces",
		index:    "compose-namespaces",
		callback: postProcNamespaces,
	}
	fmt.Errorf("blockoing error")
	return <-bErr
}

func reindexManager(ctx context.Context, log *zap.Logger, esb esutil.BulkIndexer, api *apiClient, indexPrefix string, srcQueue chan *docsSources) chan error {
	var qErr = make(chan error)
	const maxQueueLen = 3

	go func() {
		var (
			pQueueLen        = -1
			pQueueStaleCount int

			ticker = time.NewTicker(time.Second)
		)

		defer ticker.Stop()
		defer func() {
			qErr <- nil
		}()

		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != context.Canceled {
					log.Error(ctx.Err().Error())
				} else {
					log.Info("stopped")
				}
				return

			case ds := <-srcQueue:
				if ds == nil {
					// graceful termination
					log.Info("done")
					return
				}

				err := reindex(ctx, log, esb, api, indexPrefix, ds)
				if err != nil {
					log.Error("failed to reindex", zap.Error(err), zap.String("endpoint", ds.endpoint))
					return
				}

			case <-ticker.C:
				if pQueueLen != len(srcQueue) {
					pQueueStaleCount = maxQueueLen
				} else {
					pQueueStaleCount--
				}

				if pQueueStaleCount <= 0 {
					log.Info("idle")
					return
				}

				pQueueLen = len(srcQueue)

				s := esb.Stats()
				log.Debug("batch indexing stats",
					zap.Uint64("added", s.NumAdded),
					zap.Uint64("flushed", s.NumFlushed),
					zap.Uint64("failed", s.NumFailed),
					zap.Uint64("indexed", s.NumIndexed),
					zap.Uint64("requests", s.NumRequests),
					zap.Int("queue length", pQueueLen),
				)
			}
		}
	}()

	println("returning")
	return qErr
}

func reindex(ctx context.Context, log *zap.Logger, esb esutil.BulkIndexer, api *apiClient, indexPrefix string, ds *docsSources) (err error) {
	var (
		qs     = url.Values{"limit": []string{"500"}}
		req    *http.Request
		rsp    *http.Response
		cursor string
	)

	for {
		rspPayload := &rspDiscoveryDocuments{}

		if cursor != "" {
			// set new cursor and update source URL
			qs.Set("pageCursor", cursor)
		}

		if req, err = api.resources(ds.endpoint, qs); err != nil {
			return fmt.Errorf("failed to prepare request: %w", err)
		}

		if rsp, err = httpClient().Do(req.WithContext(ctx)); err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		if rsp.StatusCode != http.StatusOK {
			return fmt.Errorf("request resulted in an unexpected status '%s' for url '%s'", rsp.Status, req.URL)
		}

		//{
		//	d, err := httputil.DumpRequestOut(req, true)
		//	println(string(d))
		//	spew.Dump(err)
		//}
		//{
		//	d, err := httputil.DumpResponse(rsp, true)
		//	println(string(d))
		//	spew.Dump(err)
		//}

		if err = json.NewDecoder(rsp.Body).Decode(rspPayload); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		if err = rsp.Body.Close(); err != nil {
			return fmt.Errorf("failed to close response body: %w", err)
		}

		var docs int
		if rspPayload.Error != nil {
			log.Debug("skipping",
				zap.String("index", fmt.Sprintf(indexTpl, indexPrefix, ds.index)),
				zap.String("error", rspPayload.Error.Message),
			)
			return
		} else if rspPayload.Response != nil {
			docs = len(rspPayload.Response.Documents)
		}

		log.Debug("reindexing",
			zap.Int("docs", docs),
			zap.String("index", fmt.Sprintf(indexTpl, indexPrefix, ds.index)),
		)

		if docs == 0 {
			return
		}

		for _, doc := range rspPayload.Response.Documents {
			err = esb.Add(ctx, esutil.BulkIndexerItem{
				Index:      fmt.Sprintf(indexTpl, indexPrefix, ds.index),
				Action:     "index",
				DocumentID: doc.ID,
				Body:       bytes.NewBuffer(doc.Source),
				OnFailure: func(ctx context.Context, req esutil.BulkIndexerItem, rsp esutil.BulkIndexerResponseItem, err error) {
					spew.Dump(req)
					spew.Dump(rsp)
					spew.Dump(err)
				},
			})

			if err != nil {
				return err
			}

			if ds.callback != nil {
				go ds.callback(doc)
			}
		}

		cursor = rspPayload.Response.Filter.NextPage
		if rspPayload.Response.Filter.NextPage == "" {
			break
		}
	}

	return nil
}
