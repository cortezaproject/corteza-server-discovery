package main

import (
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/options"
	_ "github.com/joho/godotenv/autoload"
	"os"
	"strings"
)

type (
	config struct {
		es struct {
			addresses []string
		}
		cortezaAuth         string
		cortezaDiscoveryAPI string
		schemas             []*schema
	}

	schema struct {
		indexPrefix  string
		clientKey    string
		clientSecret string
	}
)

func getConfig() (*config, error) {
	c := &config{}
	return c, func() error {
		baseUrl := options.EnvString("CORTEZA_SERVER_BASE_URL", "http://server:80")

		c.cortezaAuth = options.EnvString("CORTEZA_SERVER_AUTH", baseUrl+"/auth")
		if c.cortezaAuth == "" {
			return fmt.Errorf("corteza Auth endpoint value empty, set it directly with CORTEZA_SERVER_AUTH or indirectly with CORTEZA_SERVER_BASE_URL")
		}

		c.cortezaDiscoveryAPI = options.EnvString("CORTEZA_SERVER_API_DISCOVERY", baseUrl+"/api/discovery")
		if c.cortezaDiscoveryAPI == "" {
			return fmt.Errorf("corteza Discovery API endpoint value empty, set it directly with CORTEZA_SERVER_AUTH or indirectly with CORTEZA_SERVER_API_DISCOVERY")
		}

		for _, ar := range []string{"public", "protected", "private"} {
			var (
				has  bool
				ucAr = strings.ToUpper(ar)
				s    = &schema{indexPrefix: ar}

				keyEnv = ucAr + "_INDEX_CLIENT_KEY"
				secEnv = ucAr + "_INDEX_CLIENT_SECRET"
			)

			if s.clientKey, has = os.LookupEnv(keyEnv); !has {
				continue
			} else if s.clientKey == "" {
				return fmt.Errorf("client key (%s) for '%s' is empty or missing", keyEnv, s.indexPrefix)
			}

			if s.clientSecret = os.Getenv(secEnv); s.clientSecret == "" {
				return fmt.Errorf("client secret (%s) for '%s' is empty or missing", secEnv, s.indexPrefix)
			}

			c.schemas = append(c.schemas, s)
		}

		if len(c.schemas) == 0 {
			return fmt.Errorf("set at least one client secret pair using <PREFIX>_INDEX_CLIENT_KEY and <PREFIX>_INDEX_CLIENT_SECRET where prefix is one of 'public', 'protected' or 'private'")
		}

		for _, a := range strings.Split(options.EnvString("ES_ADDRESS", "http://es:9200"), " ") {
			if a = strings.TrimSpace(a); a != "" {
				c.es.addresses = append(c.es.addresses, a)
			}
		}
		return nil
	}()
}
