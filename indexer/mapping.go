package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"go.uber.org/zap"
	"net/http"
)

type (
	esIndex struct {
		Name      string `json:"index"`
		Health    string `json:"health"`
		Status    string `json:"status"`
		DocsCount string `json:"docs.count"`
		StoreSize string `json:"store.size"`
	}

	rspDiscoveryMappings struct {
		Response []*mapping
	}

	reqMapping struct {
		// @todo settings
		Mappings struct {
			Properties map[string]*property `json:"properties"`
		} `json:"mappings"`
	}

	mapping struct {
		Index        string               `json:"index"`
		Properties   map[string]*property `json:"properties"`
		DocumentsURL string               `json:"documentsURL"`
	}

	property struct {
		// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-types.html
		Type string `json:"type,omitempty"`

		// Boost factor
		// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-boost.html
		Boost float32 `json:"boost,omitempty"`

		Properties map[string]*property `json:"properties,omitempty"`
	}
)

// fetches mappings from discovery server and
func Mappings(ctx context.Context, log *zap.Logger, esc *elasticsearch.Client, api *apiClient, indexPrefix string) (err error) {
	var (
		req             *http.Request
		rsp             *http.Response
		rspPayload      = &rspDiscoveryMappings{}
		buf             = &bytes.Buffer{}
		esRsp           *esapi.Response
		existingIndexes []*esIndex
		index           string
	)

	if req, err = api.mappings(); err != nil {
		return fmt.Errorf("failed to prepare request: %w", err)
	}

	//d, _ := httputil.DumpRequest(req, true)
	//println(string(d))

	if rsp, err = httpClient().Do(req.WithContext(ctx)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("request resulted in an unexpected status: %s", rsp.Status)
	}

	//d, _ = httputil.DumpResponse(rsp, true)
	//println(string(d))

	if err = json.NewDecoder(rsp.Body).Decode(rspPayload); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	if err = rsp.Body.Close(); err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	if existingIndexes, err = getExistingIndexes(ctx, esc); err != nil {
		return fmt.Errorf("failed to fetch existing indexes: %w", err)
	}

	indexMap := mapExistingIndexes(existingIndexes)

	for _, m := range rspPayload.Response {
		buf.Reset()
		esReq := reqMapping{}
		esReq.Mappings.Properties = m.Properties

		if err = json.NewEncoder(buf).Encode(esReq); err != nil {
			return
		}

		index = fmt.Sprintf(indexTpl, indexPrefix, m.Index)
		iLog := log.With(zap.String("name", index))

		if e := indexMap[index]; e != nil {
			iLog.Info("index exists",
				zap.String("health", e.Health),
				zap.String("status", e.Status),
				zap.String("size", e.StoreSize),
				zap.String("documents", e.DocsCount),
			)

			continue
		}

		if esRsp, err = esc.Indices.Create(index, esc.Indices.Create.WithBody(buf)); err != nil {
			iLog.Error("index creation failed", zap.Error(err))
			continue
		}

		if err = esRsp.Body.Close(); err != nil {
			return
		}

		iLog.Info("index created")
	}

	return nil
}

func mapExistingIndexes(ii []*esIndex) map[string]*esIndex {
	m := make(map[string]*esIndex)
	for _, i := range ii {
		m[i.Name] = i
	}

	return m
}

func getExistingIndexes(ctx context.Context, esc *elasticsearch.Client) (ii []*esIndex, err error) {
	var (
		esRsp *esapi.Response
	)

	ii = make([]*esIndex, 100)

	esRsp, err = esc.Cat.Indices(
		esc.Cat.Indices.WithContext(ctx),
		esc.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return
	}
	defer esRsp.Body.Close()

	return ii, json.NewDecoder(esRsp.Body).Decode(&ii)
}
