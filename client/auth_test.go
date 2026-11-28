package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientCredentials(t *testing.T) {
	var exchanges atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprint(w, `{"issuer":"https://id.example.com"}`)
		case "/api/oidc/token":
			if id, secret, _ := r.BasicAuth(); id != "id" || secret != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"error":"invalid_client"}`)
				return
			}
			if r.PostFormValue("grant_type") != "client_credentials" || r.PostFormValue("resource") != "https://id.example.com/api" {
				t.Errorf("invalid exchange: %v", r.PostForm)
			}
			fmt.Fprintf(w, `{"access_token":"t%d","token_type":"Bearer","expires_in":3600}`, exchanges.Add(1))
		default:
			if got, want := r.Header.Get("Authorization"), fmt.Sprintf("Bearer t%d", exchanges.Load()); got != want {
				t.Errorf("Authorization = %q, want %q", got, want)
			}
		}
	}))
	defer srv.Close()

	cc := NewClientCredentials(srv.URL, "id", "secret", nil)
	for range 2 {
		c, err := NewClient(srv.URL, WithClientCredentials(cc))
		if err != nil {
			t.Fatal(err)
		}
		if err := Check(c.GetUserByID(t.Context(), "u")); err != nil {
			t.Fatal(err)
		}
		cc.Invalidate()
		if err := Check(c.GetUserByID(t.Context(), "u")); err != nil {
			t.Fatal(err)
		}
	}
	if n := exchanges.Load(); n != 3 {
		t.Errorf("%d token exchanges, want 3", n)
	}

	c, _ := NewClient(srv.URL, WithClientCredentials(NewClientCredentials(srv.URL, "id", "wrong", nil)))
	if err := Check(c.GetUserByID(t.Context(), "u")); AsAPIError(err) == nil || AsAPIError(err).StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong secret: %v, want a 401 *APIError", err)
	}

	// A request waiting on another's token fetch gives up with its context.
	c, _ = NewClient(srv.URL, WithClientCredentials(cc))
	cc.lock <- struct{}{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Check(c.GetUserByID(ctx, "u")); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled wait: %v, want context.Canceled", err)
	}
}
