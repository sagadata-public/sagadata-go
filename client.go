package sagadata

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
)

const DefaultEndpoint = "https://public-api.nord-no-krs-1.sagadata.tum.fail/compute/v1"

type ClientConfig struct {
	Endpoint string
	// Token is a static API token. Either Token or TokenFile must be set.
	Token string
	// TokenFile is a path to a file containing the API token.
	// The token is read from the file on each request, allowing dynamic token updates.
	// Either Token or TokenFile must be set.
	TokenFile string
}

// newTokenFileInterceptor creates a RequestEditorFn that reads the token from a file on each request.
func newTokenFileInterceptor(tokenFile string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			return fmt.Errorf("failed to read token file %q: %w", tokenFile, err)
		}
		token := strings.TrimSpace(string(data))
		if token == "" {
			return fmt.Errorf("token file %q is empty", tokenFile)
		}
		req.Header.Add("X-Auth-Token", token)
		return nil
	}
}

func NewSagaDataClient(config ClientConfig, opts ...ClientOption) (*ClientWithResponses, error) {
	if config.Endpoint == "" {
		config.Endpoint = DefaultEndpoint
	}

	var interceptor RequestEditorFn

	switch {
	case config.Token != "" && config.TokenFile != "":
		return nil, fmt.Errorf("ClientConfig.Token and ClientConfig.TokenFile are mutually exclusive")
	case config.Token != "":
		apiKeyProvider, err := securityprovider.NewSecurityProviderApiKey("header", "X-Auth-Token", config.Token)
		if err != nil {
			return nil, err
		}
		interceptor = apiKeyProvider.Intercept
	case config.TokenFile != "":
		interceptor = newTokenFileInterceptor(config.TokenFile)
	default:
		return nil, fmt.Errorf("either ClientConfig.Token or ClientConfig.TokenFile is required")
	}

	client, err := NewClientWithResponses(config.Endpoint, append(opts, WithRequestEditorFn(interceptor))...)
	if err != nil {
		return nil, err
	}

	return client, nil
}
