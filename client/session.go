package client

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// NewSession exchanges a one-time token with remember=false and returns a client
// with a private cookie jar. End the session with Check(c.Logout(ctx)).
// httpClient may be nil; otherwise its settings are copied and its transport shared.
// Pocket ID's Secure session cookie is still sent if server is plain http.
func NewSession(ctx context.Context, server, token string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	u, err := url.Parse(server)
	if err != nil {
		return nil, err
	}
	hc := *httpClient
	jar, _ := cookiejar.New(nil) // New cannot fail with nil options.
	hc.Jar = sessionJar{jar, u.Scheme + "://" + u.Host}
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

// sessionJar treats requests to the server's exact origin as secure.
type sessionJar struct {
	*cookiejar.Jar
	origin string
}

func (j sessionJar) Cookies(u *url.URL) []*http.Cookie {
	if u.Scheme+"://"+u.Host == j.origin {
		s := *u
		s.Scheme = "https"
		u = &s
	}
	return j.Jar.Cookies(u)
}
