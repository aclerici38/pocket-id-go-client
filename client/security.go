package client

import (
	"fmt"
	"regexp"
	"slices"
)

// Security schemes Pocket ID accepts, for Scopes.
const (
	SchemeAPIKey                  = "apiKey"
	SchemeOAuth2                  = "oauth2"
	SchemeOAuth2ClientCredentials = "oauth2ClientCredentials"
	SchemeSession                 = "session"
)

// methodVariant matches the suffix of a generated method variant, such as the WithBody of
// CreateUserWithBody or the WithResponse of ListUsersWithResponse.
var methodVariant = regexp.MustCompile(`(With[A-Za-z]*Body)?(WithResponse)?$`)

// Scopes returns the scopes a credential of scheme must hold to call all the named
// operations, sorted and without duplicates. Names are client method names, such as
// "GetUserByID" or a variant like "CreateUserWithBodyWithResponse". It fails for an unknown
// name or an operation the scheme can't call.
//
// Pocket ID also requires admin-access:write for requests that decide who holds admin
// access, such as editing an admin user. That depends on the target, not the route, so
// Scopes never returns it.
func Scopes(scheme string, names ...string) ([]string, error) {
	var scopes []string
	for _, name := range names {
		var pattern string
		op := methodVariant.ReplaceAllString(name, "")
		for p, n := range operations {
			if n == op {
				pattern = p
			}
		}
		reqs, ok := security[pattern]
		if !ok {
			return nil, fmt.Errorf("pocket id: unknown operation %q", name)
		}
		s, ok := reqs[scheme]
		if !ok && reqs != nil {
			return nil, fmt.Errorf("pocket id: %s credentials can't call %s", scheme, name)
		}
		scopes = append(scopes, s...)
	}
	slices.Sort(scopes)
	return slices.Compact(scopes), nil
}
