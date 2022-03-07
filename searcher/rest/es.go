package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/go-chi/jwtauth"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type (
	// EsSearchAggrTerms is aggregations parameter for es search api.
	EsSearchAggrTerms map[string]esSearchAggr

	esSearchParamsIndex struct {
		Prefix struct {
			Index struct {
				Value string `json:"value"`
			} `json:"_index"`
		} `json:"prefix"`
	}

	esSimpleQueryString struct {
		Wrap struct {
			Query string `json:"query"`
		} `json:"simple_query_string"`
	}

	esDisMax struct {
		Wrap struct {
			TieBreaker float64       `json:"tie_breaker,omitempty"`
			Boost      float64       `json:"boost,omitempty"`
			Queries    []interface{} `json:"queries,omitempty"`
		} `json:"dis_max,omitempty"`
	}

	esMultiMatch struct {
		Wrap struct {
			Query string `json:"query"`
			Type  string `json:"type"`
			//Operator string   `json:"operator"`
			Fields []string `json:"fields"`
		} `json:"multi_match"`
	}

	esSearchParams struct {
		Query struct {
			Bool struct {
				// query context
				Must []interface{} `json:"must,omitempty"`

				// filter context
				Filter  []interface{} `json:"filter,omitempty"`
				MustNot []interface{} `json:"must_not,omitempty"`
			} `json:"bool,omitempty"`
		} `json:"query"`

		Aggregations EsSearchAggrTerms `json:"aggs,omitempty"`
	}

	esSearchAggrTerm struct {
		Field string `json:"field"`
		Size  int    `json:"size,omitempty"`
	}

	esSearchAggrComposite struct {
		Sources interface{} `json:"sources"` // it can be esSearchAggrTerm,.. (Histogram, Date histogram, GeoTile grid)
		Size    int         `json:"size,omitempty"`
	}

	esSearchAggr struct {
		Terms        esSearchAggrTerm  `json:"terms"`
		Aggregations EsSearchAggrTerms `json:"aggs,omitempty"`
		//Composite *esSearchAggrComposite `json:"composite"`
	}

	esSearchResponse struct {
		Took         int                  `json:"took"`
		TimedOut     bool                 `json:"timed_out"`
		Hits         esSearchHits         `json:"hits"`
		Aggregations esSearchAggregations `json:"aggregations"`
	}

	esSearchTotal struct {
		Value    int    `json:"value"`
		Relation string `json:"relation"`
	}

	esSearchHits struct {
		Total esSearchTotal  `json:"total"`
		Hits  []*esSearchHit `json:"hits"`
	}

	esSearchHit struct {
		Index  string          `json:"_index"`
		ID     string          `json:"_id"`
		Source json.RawMessage `json:"_source"`
	}

	esSearchAggregations struct {
		//Resource struct {
		//	DocCountErrorUpperBound int `json:"-"`
		//	SumOtherDocCount        int `json:"-"`
		//	Buckets                 []struct {
		//		Key          string `json:"key"`
		//		DocCount     int    `json:"doc_count"`
		//		ResourceName struct {
		//			DocCountErrorUpperBound int `json:"-"`
		//			SumOtherDocCount        int `json:"-"`
		//			Buckets                 []struct {
		//				Key      string `json:"key"`
		//				DocCount int    `json:"doc_count"`
		//			} `json:"buckets"`
		//		} `json:"resourceName"`
		//		Namespaces struct {
		//			DocCountErrorUpperBound int `json:"-"`
		//			SumOtherDocCount        int `json:"-"`
		//			Buckets                 []struct {
		//				Key      string `json:"key"`
		//				DocCount int    `json:"doc_count"`
		//			} `json:"buckets"`
		//		} `json:"namespaces"`
		//		Modules struct {
		//			DocCountErrorUpperBound int `json:"-"`
		//			SumOtherDocCount        int `json:"-"`
		//			Buckets                 []struct {
		//				Key      string `json:"key"`
		//				DocCount int    `json:"doc_count"`
		//			} `json:"buckets"`
		//		} `json:"modules"`
		//	} `json:"buckets"`
		//} `json:"resource"`
		Module struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"module"`
		Namespace struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"namespace"`
	}

	searchParams struct {
		query         string
		moduleAggs    []string
		namespaceAggs []string
		dumpRaw       bool
		size          int

		aggOnly  bool
		mAggOnly bool
	}
)

func esSearch(ctx context.Context, log *zap.Logger, esc *elasticsearch.Client, p searchParams) (*esSearchResponse, error) {
	var (
		buf          bytes.Buffer
		roles        []string
		userID       uint64
		_, claims, _ = jwtauth.FromContext(ctx)
	)

	if _, has := claims["roles"]; has {
		if rolesStr, is := claims["roles"].(string); is {
			roles = strings.Split(rolesStr, " ")
		}
	}
	if _, has := claims["sub"]; has {
		if sub, is := claims["sub"].(string); is {
			userID, _ = strconv.ParseUint(sub, 10, 64)
		}
	}

	noQ := len(p.query) == 0
	noNSFilter := len(p.namespaceAggs) == 0
	//noMFilter := len(p.moduleAggs) == 0
	sqs := esSimpleQueryString{}
	sqs.Wrap.Query = p.query

	query := esSearchParams{}
	index := esSearchParamsIndex{}

	// Decide what indexes we can use
	if userID == 0 {
		// Missing, invalid, expired access token (JWT)
		//index.Prefix.Index.Value = "corteza-public-"
		// fixme revert this, temp fix for searching
		index.Prefix.Index.Value = "corteza-private-"
	} else {
		// Authenticated user
		index.Prefix.Index.Value = "corteza-private-"

		//query.Query.Bool.Filter = []interface{}{
		//	//map[string]map[string]interface{}{
		//	//	"exists": {"field": []string{"security.allowedRoles", "security.deniedRoles"}},
		//	//},
		//	map[string]map[string]interface{}{
		//		// Skip all documents that do not have baring roles in the allow list
		//		"terms": {"security.allowedRoles": roles},
		//	},
		//}
		//query.Query.Bool.MustNot = []interface{}{
		//	map[string]map[string]interface{}{
		//		// Skip all documents that have baring roles in the deny list
		//		"terms": {"security.deniedRoles": roles},
		//	},
		//}
		_ = roles
	}

	// Query MUST filter
	query.Query.Bool.Must = []interface{}{index}

	// Aggregations V1.0
	//if len(p.aggregations) > 0 {
	//	query.Aggregations = make(map[string]esSearchAggr)
	//
	//	for _, a := range p.aggregations {
	//		query.Aggregations[a] = esSearchAggr{esSearchAggrTerm{Field: a + ".keyword"}}
	//	}
	//}

	// Search string filter
	if !noQ {
		sqs.Wrap.Query = fmt.Sprintf("%s*", sqs.Wrap.Query)
		query.Query.Bool.Must = append(query.Query.Bool.Must, sqs)
		//query.Query.DisMax.Queries = append(query.Query.DisMax.Queries, sqs)
	}

	var (
		mm = esMultiMatch{}
		dd esDisMax
	)
	for _, mAggs := range p.moduleAggs {
		mm.Wrap.Query = mAggs
		mm.Wrap.Type = "cross_fields"
		mm.Wrap.Fields = []string{"module.name.keyword"}
		//query.Query.Bool.Must = append(query.Query.Bool.Must, mm)
		//query.Query.DisMax.Queries = append(query.Query.DisMax.Queries, mm)

		dd.Wrap.Queries = append(dd.Wrap.Queries, mm)
	}

	// no need now since we are adding below as filter
	if p.aggOnly {
		for _, nAggs := range p.namespaceAggs {
			mm.Wrap.Query = nAggs
			mm.Wrap.Type = "cross_fields"
			mm.Wrap.Fields = []string{"namespace.name.keyword"}
			//query.Query.Bool.Must = append(query.Query.Bool.Must, mm)
			//query.Query.DisMax.Queries = append(query.Query.DisMax.Queries, mm)

			dd.Wrap.Queries = append(dd.Wrap.Queries, mm)
		}
	}

	if len(dd.Wrap.Queries) > 0 {
		query.Query.Bool.Must = append(query.Query.Bool.Must, dd)
	}

	if !p.aggOnly && !noNSFilter {
		nsf := make(map[string]interface{})
		nsf["terms"] = map[string][]string{
			"namespace.name.keyword": p.namespaceAggs,
		}
		query.Query.Bool.Filter = append(query.Query.Bool.Filter, nsf)
	}

	// Aggregations V1.0 Improved
	//if len(p.aggregations) > 0 {
	//	for _, a := range p.aggregations {
	//		if len(a) > 0 {
	//			sqs = esSimpleQueryString{}
	//			sqs.Wrap.Query = a
	//			query.Query.Bool.Must = append(query.Query.Bool.Must, sqs)
	//		}
	//	}
	//}

	//if noQ == 0 && len(p.moduleAggs) == 0 && len(p.namespaceAggs) == 0 {
	//	query.Query.DisMax.Queries = append(query.Query.DisMax.Queries, index)
	//}
	query.Aggregations = make(map[string]esSearchAggr)
	query.Aggregations["namespace"] = esSearchAggr{
		Terms: esSearchAggrTerm{
			Field: "namespace.name.keyword",
			Size:  999,
		},
	}

	if !noQ || !noNSFilter {
		query.Aggregations["module"] = esSearchAggr{
			Terms: esSearchAggrTerm{
				Field: "module.name.keyword",
				Size:  999,
			},
		}
	}

	//query.Aggregations["resource"] = esSearchAggr{
	//	Terms: esSearchAggrTerm{
	//		Field: "resourceType.keyword",
	//		Size:  999,
	//	},
	//	Aggregations: EsSearchAggrTerms{
	//		"resourceName": esSearchAggr{
	//			Terms: esSearchAggrTerm{
	//				Field: "name.keyword",
	//				Size:  999,
	//			},
	//		},
	//		"modules": esSearchAggr{
	//			Terms: esSearchAggrTerm{
	//				Field: "module.name.keyword",
	//				Size:  999,
	//			},
	//		},
	//		"namespaces": esSearchAggr{
	//			Terms: esSearchAggrTerm{
	//				Field: "namespace.name.keyword",
	//				Size:  999,
	//			},
	//		},
	//	},
	//}

	// Aggregations V2.0
	//if len(p.aggregations) > 0 {
	//	query.Aggregations = (Aggregations{}).encodeTerms(p.aggregations)
	//}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("could not encode query: %q", err)
	}

	// Why set size to 999? default value for size is 10,
	// so we needed to set value till we add (@todo) pagination to search result
	if p.size == 0 {
		p.size = 999
	}

	sReqArgs := []func(*esapi.SearchRequest){
		esc.Search.WithContext(ctx),
		esc.Search.WithBody(&buf),
		esc.Search.WithTrackTotalHits(true),
		//esc.Search.WithScroll(),
		esc.Search.WithSize(p.size),
		//esc.Search.WithFrom(0), // paging (offset)
		//esc.Search.WithExplain(true), // debug
	}

	if p.dumpRaw {
		sReqArgs = append(
			sReqArgs,
			esc.Search.WithSourceExcludes("security"),
			esc.Search.WithPretty(),
		)
	}

	// Perform the search request.
	res, err := esc.Search(sReqArgs...)

	if err != nil {
		return nil, err
	}

	if err = validElasticResponse(res, err); err != nil {
		return nil, fmt.Errorf("invalid search response: %w", err)
	}

	defer res.Body.Close()

	if p.dumpRaw {
		// Copy body buf and then restore it
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		os.Stdout.Write(bodyBytes)
	}

	var sr = &esSearchResponse{}
	if err = json.NewDecoder(res.Body).Decode(sr); err != nil {
		return nil, err
	}

	// Print the response status, number of results, and request duration.
	log.Debug("search completed",
		zap.String("query", sqs.Wrap.Query),
		zap.String("indexPrefix", index.Prefix.Index.Value),
		zap.String("status", res.Status()),
		zap.Int("took", sr.Took),
		zap.Bool("timedOut", sr.TimedOut),
		zap.Int("hits", sr.Hits.Total.Value),
		zap.String("hitsRelation", sr.Hits.Total.Relation),
		zap.Int("namespaceAggs", len(sr.Aggregations.Namespace.Buckets)),
		zap.Int("moduleAggs", len(sr.Aggregations.Module.Buckets)),
	)

	return sr, nil
}

// @todo move this to es service
func validElasticResponse(res *esapi.Response, err error) error {
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
