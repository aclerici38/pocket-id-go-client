package client

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestOIDCClientID(t *testing.T) {
	for _, id := range []string{"grafana", "my-client.id_2", "https://apps.example.com/client.json?x=1&y=2#fragment", "http://apps.example.com/client.json", "ftp://apps.example.com/client.json"} {
		want := "~" + base64.RawURLEncoding.EncodeToString([]byte(id))
		got, err := OIDCClientID(id).MarshalText()
		if err != nil || string(got) != want {
			t.Errorf("%q: text=%q, want=%q, err=%v", id, got, want, err)
		}
	}
}

func TestOIDCClientIDRequests(t *testing.T) {
	const server = "https://id.example.com"
	for _, id := range []OIDCClientID{"grafana", "https://apps.example.com/client.json?x=1#fragment"} {
		segment, _ := id.MarshalText()
		for _, tc := range []struct {
			path string
			new  func() (*http.Request, error)
		}{
			{"/api/admin/oidc/clients/" + string(segment), func() (*http.Request, error) { return NewGetOIDCClientRequest(server, id) }},
			{"/api/admin/oidc/clients/" + string(segment) + "/secrets", func() (*http.Request, error) { return NewListClientSecretsRequest(server, id) }},
			{"/api/admin/api-access/" + string(segment) + "/apis", func() (*http.Request, error) { return NewListAPIsClientMayAccessRequest(server, id) }},
		} {
			req, err := tc.new()
			if err != nil {
				t.Fatal(err)
			}
			if req.URL.EscapedPath() != tc.path || req.URL.RawQuery != "" || req.URL.Fragment != "" {
				t.Errorf("request URL = %s, want %s%s", req.URL, server, tc.path)
			}
		}
	}
}
