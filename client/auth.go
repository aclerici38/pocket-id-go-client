package client

import (
	"context"
	"net/http"
)

// APIKeyHeader is the header Pocket ID reads API keys from (the spec's apiKey security scheme).
const APIKeyHeader = "X-API-Key"

// WithAPIKey authenticates every request with a Pocket ID API key.
func WithAPIKey(key string) ClientOption {
	return WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set(APIKeyHeader, key)
		return nil
	})
}
