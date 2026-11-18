package client

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Plain http uses a non-loopback host: cookiejar treats loopback as secure.
func TestNewSession(t *testing.T) {
	for _, tc := range []string{"https/session_token", "https/__Host-session_token", "http/session_token", "http/__Host-session_token"} {
		scheme, name, _ := strings.Cut(tc, "/")
		t.Run(tc, func(t *testing.T) {
			var loggedOut string
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(APIKeyHeader) != "" {
					t.Error("session sent an API key")
				}
				switch r.URL.Path {
				case "/prefix/api/one-time-access-token":
					var body struct{ Token string }
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if r.Method != http.MethodPost || r.URL.Query().Get("remember") != "false" || r.Header.Get("Cookie") != "" {
						t.Errorf("invalid exchange: %s %s; cookies %q", r.Method, r.URL, r.Header.Get("Cookie"))
					}
					if body.Token == "bad" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					http.SetCookie(w, &http.Cookie{Name: name, Value: body.Token, Path: "/", Secure: true})
				case "/prefix/api/webauthn/logout":
					loggedOut = r.Header.Get("Cookie")
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s", r.URL)
				}
			}))
			defer srv.Close()
			var hc *http.Client
			base := "http://pocket-id.svc/prefix"
			if scheme == "https" {
				srv.StartTLS()
				hc, base = srv.Client(), srv.URL+"/prefix"
			} else {
				srv.Start()
				hc = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
				}}}
			}
			hc.Timeout = time.Second
			hc.Jar, _ = cookiejar.New(nil)
			a, err := NewSession(t.Context(), base, "a", hc)
			if err != nil {
				t.Fatal(err)
			}
			b, err := NewSession(t.Context(), base, "b", hc)
			if err != nil {
				t.Fatal(err)
			}
			copy := a.Client.(*http.Client)
			if copy == hc || copy.Jar == hc.Jar || copy.Transport != hc.Transport || copy.Timeout != hc.Timeout {
				t.Fatal("session did not copy settings and isolate its jar")
			}
			for token, c := range map[string]*Client{"a": a, "b": b} {
				if err := Check(c.Logout(t.Context())); err != nil || loggedOut != name+"="+token {
					t.Fatalf("logout: %v; cookie %q", err, loggedOut)
				}
			}
			if _, err := NewSession(t.Context(), base, "bad", hc); AsAPIError(err) == nil || AsAPIError(err).StatusCode != http.StatusUnauthorized {
				t.Fatalf("failed exchange: %v", err)
			}
		})
	}
}

func TestSessionJarOrigin(t *testing.T) {
	for origin, other := range map[string]string{"https://p.test": "http://p.test", "http://p.test": "http://p.test:8080"} {
		jar, _ := cookiejar.New(nil)
		j := sessionJar{jar, origin}
		u, _ := url.Parse(origin)
		o, _ := url.Parse(other)
		j.SetCookies(u, []*http.Cookie{{Name: "s", Value: "v", Secure: true}})
		if len(j.Cookies(u)) != 1 || len(j.Cookies(o)) != 0 {
			t.Errorf("%s: secure cookie not limited to its origin", origin)
		}
	}
}
