package client

import (
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"testing"
)

func TestOperationName(t *testing.T) {
	for _, tt := range []struct{ method, path, name, route string }{
		{"GET", "/api/admin/users/u1", "GetUserByID", "/api/admin/users/{id}"},
		// A literal route wins over a wildcard one.
		{"GET", "/api/users/me", "GetCurrentUser", "/api/users/me"},
		{"GET", "/api/admin/oidc/clients/~YWJj/secrets", "ListClientSecrets", "/api/admin/oidc/clients/{id}/secrets"},
		{"POST", "/api/webauthn/logout", "Logout", "/api/webauthn/logout"},
		{"GET", "/prefix/api/admin/users/u1", "", ""},
		{"PURGE", "/api/admin/users/u1", "", ""},
	} {
		req, _ := http.NewRequest(tt.method, "https://id.example.com"+tt.path, nil)
		if name, route := OperationName(req); name != tt.name || route != tt.route {
			t.Errorf("%s %s: got %q %q, want %q %q", tt.method, tt.path, name, route, tt.name, tt.route)
		}
	}
}

// Every client method has a route, so the table can't silently miss operations if the
// comments it is read from change.
func TestOperationsCoverClient(t *testing.T) {
	names := slices.Collect(maps.Values(operations))
	// Operations with a body also get variants such as CreateAPIWithBody.
	variant := regexp.MustCompile(`With[A-Za-z]*Body$`)
	for m := range reflect.TypeFor[ClientInterface]().Methods() {
		if name := variant.ReplaceAllString(m.Name, ""); !slices.Contains(names, name) {
			t.Errorf("no route for %s", name)
		}
	}
}
