package client

import "encoding/base64"

// OIDCClientID is a raw OIDC client ID used as a path parameter, not a JSON field.
type OIDCClientID string

// MarshalText encodes the ID as a single path segment decoded by Pocket ID.
func (id OIDCClientID) MarshalText() ([]byte, error) {
	return []byte("~" + base64.RawURLEncoding.EncodeToString([]byte(id))), nil
}
