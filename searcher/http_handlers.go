package searcher

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx/types"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var _ = spew.Dump

type (
	handlers struct {
		log *zap.Logger
		esc *elasticsearch.Client
		api apiClientService
	}

	cResponse struct {
		Response struct {
			Set []struct {
				NamespaceID uint64 `json:",string"`
				Slug        string `json:"slug"`

				Name     string         `json:"name"`
				ModuleID uint64         `json:",string"`
				Handle   string         `json:"handle"`
				Meta     types.JSONText `json:"meta"`
			} `json:"set,omitempty"`
		} `json:"response,omitempty"`
	}

	moduleMeta struct {
		Discovery ModuleMeta `json:"discovery"`
	}

	ModuleMeta struct {
		Public struct {
			Result []Result `json:"result"`
		} `json:"public"`
		Private struct {
			Result []Result `json:"result"`
		} `json:"private"`
		Protected struct {
			Result []Result `json:"result"`
		} `json:"protected"`
	}

	Result struct {
		Lang   string   `json:"lang"`
		Fields []string `json:"fields"`
		// @todo? TBD? excludeModuleFields, includeModuleFields <- if passed filter module field accordingly.
	}
)

//func (m moduleMeta) Read(p []byte) (n int, err error) {
//	panic("implement me")
//}

func Handlers(r chi.Router, log *zap.Logger, esc *elasticsearch.Client, api apiClientService) *handlers {
	h := &handlers{
		esc: esc,
		log: log,
		api: api,
	}

	r.Use()

	r.Get("/healthcheck", h.Healthcheck)
	r.Get("/sandbox", h.Sandbox)
	r.Get("/", h.Search)
	//r.Get("/suggest", h.Suggest)

	return h
}

func (h handlers) Healthcheck(w http.ResponseWriter, r *http.Request) {
	res, err := h.esc.Ping(
		h.esc.Ping.WithContext(r.Context()),
	)

	if validElasticResponse(h.log, res, err) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "unhealthy")
		return
	}

	defer res.Body.Close()

	_, _ = fmt.Fprintf(w, "healthy")
}

func (h handlers) Sandbox(w http.ResponseWriter, r *http.Request) {
	p := "." + r.URL.Path
	if p == "./" {
		p = "./sandbox/index.html"
	}
	http.ServeFile(w, r, p)
}

func (h handlers) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = r.ParseForm()

	size := 0
	if len(r.FormValue("size")) > 0 {
		size, _ = strconv.Atoi(r.FormValue("size"))
	}

	//searchString := r.FormValue("q")
	var (
		ctx = r.Context()
		// fixme cleanup make struct or something
		searchString  = r.FormValue("q")
		moduleAggs    = r.Form["moduleAggs"]
		namespaceAggs = r.Form["namespaceAggs"]

		results       *esSearchResponse
		aggregation   *esSearchResponse
		nsAggregation *esSearchResponse
		mAggregation  *esSearchResponse
		err           error

		nsReq      *http.Request
		nsRes      *http.Response
		mReq       *http.Request
		mRes       *http.Response
		nsResponse cResponse
		mResponse  cResponse
		moduleMap  = make(map[string][]string)

		nsHandleMap = make(map[string]string)
		mHandleMap  = make(map[string]string)
	)
	results, err = search(ctx, h.esc, h.log, searchParams{
		query:         searchString,
		moduleAggs:    moduleAggs,
		namespaceAggs: namespaceAggs,
		size:          size,
		dumpRaw:       r.FormValue("dump") != "",
	})

	if err != nil {
		h.log.Error("could not execute search", zap.Error(err))
	}

	if len(searchString) == 0 {
		aggregation, err = search(ctx, h.esc, h.log, searchParams{
			size:          size,
			dumpRaw:       r.FormValue("dump") != "",
			namespaceAggs: namespaceAggs,
			aggOnly:       true,
		})
		if err != nil {
			h.log.Error("could not execute aggregation search", zap.Error(err))
		}
	}

	// append all namespace agg with counts no matter what
	nsAggregation, err = search(ctx, h.esc, h.log, searchParams{
		size:    size,
		dumpRaw: r.FormValue("dump") != "",
		aggOnly: true,
	})
	if err != nil {
		h.log.Error("could not execute aggregation search", zap.Error(err))
	}
	if len(searchString) == 0 {
		if aggregation != nil && nsAggregation != nil {
			aggregation.Aggregations.Namespace = nsAggregation.Aggregations.Namespace
		}
	} else {
		//if results != nil && nsAggregation != nil {
		//	nsMap := make(map[string]struct {
		//		Key      string `json:"key"`
		//		DocCount int    `json:"doc_count"`
		//	})
		//	for _, bucket := range results.Aggregations.Namespace.Buckets {
		//		nsMap[bucket.Key] = bucket
		//	}
		//
		//	var buckets []struct {
		//		Key      string `json:"key"`
		//		DocCount int    `json:"doc_count"`
		//	}
		//	for _, bucket := range nsAggregation.Aggregations.Namespace.Buckets {
		//		val, ok := nsMap[bucket.Key]
		//		if ok {
		//			val.DocCount = nsMap[bucket.Key].DocCount
		//		} else {
		//			val.Key = bucket.Key
		//			val.DocCount = 0
		//		}
		//		buckets = append(buckets, val)
		//	}
		//
		//	results.Aggregations.Namespace.Buckets = buckets
		//}
	}
	// append namespace agg response which are not in es response
	if results != nil && len(namespaceAggs) > 0 {
		nsMap := make(map[string]struct {
			Key      string `json:"key"`
			DocCount int    `json:"doc_count"`
		})
		var bb []struct {
			Key      string `json:"key"`
			DocCount int    `json:"doc_count"`
		}
		for _, b := range results.Aggregations.Namespace.Buckets {
			nsMap = map[string]struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			}{
				b.Key: b,
			}
			bb = append(bb, b)
		}

		for _, agg := range namespaceAggs {
			if _, ok := nsMap[agg]; !ok {
				nsMap = map[string]struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				}{
					agg: {Key: agg, DocCount: 0},
				}
				bb = append(bb, struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				}{Key: agg, DocCount: 0})
			}
		}

		if len(bb) > 0 {
			results.Aggregations.Namespace.Buckets = bb
		}
	}

	mAggregation, err = search(ctx, h.esc, h.log, searchParams{
		size:          size,
		dumpRaw:       r.FormValue("dump") != "",
		query:         searchString,
		namespaceAggs: namespaceAggs,
		aggOnly:       true,
		mAggOnly:      true,
	})
	if err != nil {
		h.log.Error("could not execute aggregation search", zap.Error(err))
	}
	if len(searchString) > 0 {
		if results != nil && mAggregation != nil {
			results.Aggregations.Module = mAggregation.Aggregations.Module
		}
	}

	// append module agg response which are not in es response
	if results != nil && len(moduleAggs) > 0 {
		mMap := make(map[string]struct {
			Key      string `json:"key"`
			DocCount int    `json:"doc_count"`
		})
		var bb []struct {
			Key      string `json:"key"`
			DocCount int    `json:"doc_count"`
		}
		for _, b := range results.Aggregations.Module.Buckets {
			mMap = map[string]struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			}{
				b.Key: b,
			}
			bb = append(bb, b)
		}

		for _, agg := range moduleAggs {
			if _, ok := mMap[agg]; !ok {
				mMap = map[string]struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				}{
					agg: {Key: agg, DocCount: 0},
				}
				bb = append(bb, struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				}{Key: agg, DocCount: 0})
			}
		}

		if len(bb) > 0 {
			results.Aggregations.Module.Buckets = bb
		}
	}

	noHits := len(searchString) == 0 && len(moduleAggs) == 0 && len(namespaceAggs) == 0
	//if !noHits {
	// @todo only fetch module from result but that requires another loop to fetch module Id from es response
	// 			TEMP fix, I have solution use elastic for the same but different index
	nsReq, err = h.api.Namespaces()
	if err != nil {
		h.log.Warn("failed to prepare namespace request: %w", zap.Error(err))
	} else {
		if nsRes, err = h.api.HttpClient().Do(nsReq.WithContext(ctx)); err != nil {
			h.log.Error("failed to send namespace request: %w", zap.Error(err))
		}
		if nsRes.StatusCode != http.StatusOK {
			fmt.Println("err: ", err)
			h.log.Error("request resulted in an unexpected status: %s", zap.Error(err))
		}
		if err = json.NewDecoder(nsRes.Body).Decode(&nsResponse); err != nil {
			h.log.Error("failed to decode response: %w", zap.Error(err))
		}
		if err = nsRes.Body.Close(); err != nil {
			h.log.Error("failed to close response body: %w", zap.Error(err))
		}

		for _, s := range nsResponse.Response.Set {
			// Get the module handles for aggs response
			nsHandleMap[s.Name] = s.Slug
			if mReq, err = h.api.Modules(s.NamespaceID); err != nil {
				h.log.Error("failed to prepare module meta request: %w", zap.Error(err))
			}
			if mRes, err = h.api.HttpClient().Do(mReq.WithContext(ctx)); err != nil {
				h.log.Error("failed to send module request: %w", zap.Error(err))
			}
			if mRes.StatusCode != http.StatusOK {
				h.log.Error("request resulted in an unexpected status: %s", zap.Error(err))
			}
			if err = json.NewDecoder(mRes.Body).Decode(&mResponse); err != nil {
				h.log.Error("failed to decode response: %w", zap.Error(err))
			}
			if err = mRes.Body.Close(); err != nil {
				h.log.Error("failed to close response body: %w", zap.Error(err))
			}

			for _, m := range mResponse.Response.Set {
				// Get the module handles for aggs response
				mHandleMap[m.Name] = m.Handle
				var (
					meta moduleMeta
					key  = fmt.Sprintf("%d-%d", s.NamespaceID, m.ModuleID)
				)
				err = json.Unmarshal(m.Meta, &meta)
				if err != nil {
					h.log.Error("failed to unmarshal module meta: %w", zap.Error(err))
				} else if len(meta.Discovery.Private.Result) > 0 && len(meta.Discovery.Private.Result[0].Fields) > 0 {
					moduleMap[key] = meta.Discovery.Private.Result[0].Fields
				}
			}
		}
	}
	//}

	if cres, err := conv(results, aggregation, noHits, moduleMap, nsHandleMap, mHandleMap); err != nil {
		h.log.Error("could not encode response body", zap.Error(err))
	} else if err = json.NewEncoder(w).Encode(cres); err != nil {
		h.log.Error("could not encode response body", zap.Error(err))
	}
}
