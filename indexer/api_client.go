package indexer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type (
	credentials struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		expiresAt   time.Time

		authBaseUri string
		key         string
		secret      string
	}

	apiClient struct {
		baseUri     string
		credentials *credentials
	}
)

func httpClient() *http.Client {
	return http.DefaultClient
}

func ApiClient(apiBaseUri, authBaseUri, key, secret string) (c *apiClient, err error) {
	c = &apiClient{baseUri: apiBaseUri}
	c.credentials = &credentials{authBaseUri: authBaseUri, key: key, secret: secret}
	return c, err
}

func (c *apiClient) mappings() (*http.Request, error) {
	return c.request(c.baseUri + "/mappings/")
}

func (c *apiClient) resources(endpoint string, qs url.Values) (*http.Request, error) {
	return c.request(c.baseUri + "/resources/" + strings.TrimLeft(endpoint, "/") + "?" + qs.Encode())
}

func (c *apiClient) request(endpoint string) (req *http.Request, err error) {
	if err = c.authenticate(); err != nil {
		return
	}

	if req, err = http.NewRequest(http.MethodGet, endpoint, nil); err != nil {
		return
	}

	req.Header.Set("User-Agent", "corteza-discovery-indexer/0.1")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.credentials.AccessToken))
	return
}

func (c *apiClient) authenticate() (err error) {
	if c.credentials == nil {
		return fmt.Errorf("missing credentials")
	}

	if c.credentials.expiresAt.Before(time.Now()) {
		c.credentials, err = authenticate(c.credentials.authBaseUri, c.credentials.key, c.credentials.secret)
		if err != nil {
			return
		}
	}

	return nil
}

func authenticate(authBaseUri, key, secret string) (crd *credentials, err error) {
	var (
		req  *http.Request
		rsp  *http.Response
		form = url.Values{}
	)

	form.Set("grant_type", "client_credentials")
	form.Set("scope", "discovery")

	req, err = http.NewRequest(http.MethodPost, authBaseUri+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return
	}

	req.SetBasicAuth(key, secret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	//d, _ := httputil.DumpRequest(req, true)
	//println(string(d))

	rsp, err = httpClient().Do(req)
	if err != nil {
		return
	}

	defer rsp.Body.Close()
	crd = &credentials{
		authBaseUri: authBaseUri,
		key:         key,
		secret:      secret,
	}

	if rsp.StatusCode != http.StatusOK {
		aux := struct{ Error string }{}
		if err = json.NewDecoder(rsp.Body).Decode(&aux); err != nil {
			return
		} else if aux.Error != "" {
			return nil, fmt.Errorf(aux.Error)
		} else {
			return nil, fmt.Errorf("can not authenticate, unexpected error")
		}

	}

	//d, _ := httputil.DumpResponse(rsp, true)
	//println(string(d))

	err = json.NewDecoder(rsp.Body).Decode(crd)
	if err != nil {
		return
	}

	crd.expiresAt = time.Now().Add(time.Second * time.Duration(crd.ExpiresIn))

	return
}
