package client

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	for _, name := range []string{"session_token", "__Host-session_token"} {
		t.Run(name, func(t *testing.T) {
			var loggedOut string
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			hc := srv.Client()
			hc.Timeout = time.Second
			hc.Jar, _ = cookiejar.New(nil)
			base := srv.URL + "/prefix"
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
