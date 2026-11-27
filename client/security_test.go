package client

import (
	"maps"
	"slices"
	"testing"
)

func TestScopes(t *testing.T) {
	for _, tt := range []struct {
		scheme string
		names  []string
		want   []string
		err    bool
	}{
		{SchemeAPIKey, []string{"ListUsers", "CreateUser", "GetUserByID"}, []string{"users:read", "users:write"}, false},
		{SchemeOAuth2ClientCredentials, []string{"ListUserGroups"}, []string{"groups:read"}, false},
		{SchemeAPIKey, []string{"GetCurrentDeployedVersionOfPocketID", "RespondsToHealthchecks"}, nil, false},
		{SchemeSession, []string{"Logout"}, nil, false},
		{SchemeAPIKey, []string{"CreateAPIKey"}, nil, true},
		{SchemeAPIKey, []string{"Logout"}, nil, true},
		{SchemeAPIKey, []string{"CreateUserWithBody", "GetUserByIDWithResponse"}, []string{"users:read", "users:write"}, false},
		{SchemeAPIKey, []string{"IntrospectOIDCTokensWithFormdataBodyWithResponse"}, nil, false},
		{SchemeAPIKey, []string{"Bogus"}, nil, true},
	} {
		got, err := Scopes(tt.scheme, tt.names...)
		if (err != nil) != tt.err || !slices.Equal(got, tt.want) {
			t.Errorf("Scopes(%q, %q) = %q, %v", tt.scheme, tt.names, got, err)
		}
	}
}

// The tables come from different sources, so check they cover the same routes.
func TestSecurityCoversOperations(t *testing.T) {
	if got, want := slices.Sorted(maps.Keys(security)), slices.Sorted(maps.Keys(operations)); !slices.Equal(got, want) {
		t.Errorf("security routes %q\ndon't match operations %q", got, want)
	}
}
