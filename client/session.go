package client

import (
	"context"
	"net/http"
	"net/http/cookiejar"
)

// NewSession exchanges a one-time token with remember=false and returns a client
// with a private cookie jar. End the session with Check(c.Logout(ctx)).
// httpClient may be nil; otherwise its settings are copied and its transport shared.
func NewSession(ctx context.Context, server, token string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	hc := *httpClient
	hc.Jar, _ = cookiejar.New(nil) // New cannot fail with nil options.
	c, err := NewClient(server, WithHTTPClient(&hc))
	if err != nil {
		return nil, err
	}
	if err := Check(c.ExchangeOneTimeAccessToken(ctx,
		&ExchangeOneTimeAccessTokenParams{Remember: new(false)},
		ExchangeOneTimeAccessTokenJSONRequestBody{Token: &token},
	)); err != nil {
		return nil, err
	}
	return c, nil
}
