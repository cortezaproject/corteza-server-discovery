package options

import (
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/options"
	"os"
	"strings"
)

type (
	IndexerOpt struct {
		HttpAddr             string
		CortezaAuth          string
		CortezaServerBaseUrl string
		CortezaDiscoveryAPI  string

		Schemas []*schema
	}

	schema struct {
		IndexPrefix  string
		ClientKey    string
		ClientSecret string
	}
)

const (
	envKeyHttpAddr = "HTTP_ADDR"
)

func Indexer() (o *IndexerOpt, err error) {
	o = &IndexerOpt{}
	return o, func() error {
		baseUrl := options.EnvString("CORTEZA_SERVER_BASE_URL", "http://server:80")

		o.HttpAddr = options.EnvString(envKeyHttpAddr, "127.0.0.1:3201")

		o.CortezaAuth = options.EnvString("CORTEZA_SERVER_AUTH", baseUrl+"/auth")
		if o.CortezaAuth == "" {
			return fmt.Errorf("corteza Auth endpoint value empty, set it directly with CORTEZA_SERVER_AUTH or indirectly with CORTEZA_SERVER_BASE_URL")
		}

		o.CortezaDiscoveryAPI = options.EnvString("CORTEZA_SERVER_API_DISCOVERY", baseUrl+"/api/discovery")
		if o.CortezaDiscoveryAPI == "" {
			return fmt.Errorf("corteza Discovery API endpoint value empty, set it directly with CORTEZA_SERVER_AUTH or indirectly with CORTEZA_SERVER_API_DISCOVERY")
		}

		for _, ar := range []string{"public", "protected", "private"} {
			var (
				has  bool
				ucAr = strings.ToUpper(ar)
				s    = &schema{IndexPrefix: ar}

				keyEnv = ucAr + "_INDEX_CLIENT_KEY"
				secEnv = ucAr + "_INDEX_CLIENT_SECRET"
			)

			if s.ClientKey, has = os.LookupEnv(keyEnv); !has {
				continue
			} else if s.ClientKey == "" {
				return fmt.Errorf("client key (%s) for '%s' is empty or missing", keyEnv, s.IndexPrefix)
			}

			if s.ClientSecret = os.Getenv(secEnv); s.ClientSecret == "" {
				return fmt.Errorf("client secret (%s) for '%s' is empty or missing", secEnv, s.IndexPrefix)
			}

			o.Schemas = append(o.Schemas, s)
		}

		if len(o.Schemas) == 0 {
			return fmt.Errorf("set at least one client secret pair using <PREFIX>_INDEX_CLIENT_KEY and <PREFIX>_INDEX_CLIENT_SECRET where prefix is one of 'public', 'protected' or 'private'")
		}

		return nil
	}()
}
