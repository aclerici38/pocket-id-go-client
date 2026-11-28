package client

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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

// ClientCredentials holds an OIDC client's access token for Pocket ID's own API. Share one
// across clients to share the token, which is fetched with the context of the request that
// needs it.
type ClientCredentials struct {
	server string
	hc     *http.Client
	cfg    clientcredentials.Config
	lock   chan struct{} // see acquire
	tok    *oauth2.Token
}

// NewClientCredentials authenticates as an OIDC client with the client credentials grant.
// The client must be granted access to the Pocket ID API on its own behalf. Without scopes,
// tokens hold every scope it's granted. httpClient may be nil.
func NewClientCredentials(server, clientID, secret string, httpClient *http.Client, scopes ...string) *ClientCredentials {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ClientCredentials{server: server, hc: httpClient, lock: make(chan struct{}, 1), cfg: clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		TokenURL:     strings.TrimRight(server, "/") + "/api/oidc/token",
		Scopes:       scopes,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}}
}

// Invalidate drops the cached token. Tokens keep the scopes they were issued with, so call it
// after widening the client's grants. A token rejected with a 401, such as once the client
// loses a scope it carries, is dropped without it.
func (cc *ClientCredentials) Invalidate() {
	cc.lock <- struct{}{}
	cc.tok = nil
	<-cc.lock
}

// WithClientCredentials authenticates every request with a token from cc. Pass it after
// WithHTTPClient: it wraps the client's doer, or else cc's http.Client, to drop the token when
// Pocket ID rejects it.
func WithClientCredentials(cc *ClientCredentials) ClientOption {
	return func(c *Client) error {
		c.Client = rejectedTokens{cmp.Or[HttpRequestDoer](c.Client, cc.hc), cc}
		return WithRequestEditorFn(cc.authorize)(c)
	}
}

// rejectedTokens drops cc's token when a request gets a 401. A request still carrying an older
// token may drop a newer one, which costs only an extra token request.
type rejectedTokens struct {
	next HttpRequestDoer
	cc   *ClientCredentials
}

func (d rejectedTokens) Do(req *http.Request) (*http.Response, error) {
	rsp, err := d.next.Do(req)
	if err == nil && rsp.StatusCode == http.StatusUnauthorized && d.cc.acquire(req.Context()) == nil {
		d.cc.tok = nil
		<-d.cc.lock
	}
	return rsp, err
}

// acquire takes cc's lock, a mutex whose waiters give up when ctx ends.
func (cc *ClientCredentials) acquire(ctx context.Context) error {
	select {
	case cc.lock <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (cc *ClientCredentials) authorize(ctx context.Context, req *http.Request) error {
	if err := cc.acquire(ctx); err != nil {
		return err
	}
	defer func() { <-cc.lock }()
	if !cc.tok.Valid() {
		tok, err := cc.fetch(ctx)
		if err != nil {
			return fmt.Errorf("client credentials token: %w", err)
		}
		cc.tok = tok
	}
	cc.tok.SetAuthHeader(req)
	return nil
}

// fetch requests a token for the issuer's API, its audience even if server is internal.
// Failed requests return an *APIError.
func (cc *ClientCredentials) fetch(ctx context.Context) (*oauth2.Token, error) {
	c, err := NewClient(cc.server, WithHTTPClient(cc.hc))
	if err != nil {
		return nil, err
	}
	d, err := Decode[struct{ Issuer string }](c.GetOpenIDConnectDiscoveryConfiguration(ctx))
	if err != nil {
		return nil, err
	}
	cfg := cc.cfg
	cfg.EndpointParams = url.Values{"resource": {d.Issuer + "/api"}}
	tok, err := cfg.Token(context.WithValue(ctx, oauth2.HTTPClient, cc.hc))
	if re, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		re.Response.Body = io.NopCloser(bytes.NewReader(re.Body))
		// A 2xx with an error body keeps the oauth2 error.
		if _, apiErr := read(re.Response, nil); apiErr != nil {
			err = apiErr
		}
	}
	return tok, err
}
