# pocket-id-go-client

Go client for [Pocket ID](https://github.com/pocket-id/pocket-id), generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) from the OpenAPI 3.1 spec that upstream builds from its route definitions.

```sh
go get github.com/aclerici38/pocket-id-go-client/v2
```

Release tags match Pocket ID versions, and the module's major version follows Pocket ID's.

## Authentication

Pass an API key with `WithAPIKey`, which sends it in the `X-API-Key` header:

```go
c, err := client.NewClientWithResponses("https://id.example.com", client.WithAPIKey(key))
```

For OAuth access tokens, give `WithHTTPClient` an `*http.Client` from `golang.org/x/oauth2`.

For temporary access as a user, exchange a one-time access token with `NewSession`:

```go
c, err := client.NewSession(ctx, "https://id.example.com", oneTimeToken, nil)
if err != nil {
	return err
}
defer func() {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := client.Check(c.Logout(cleanupCtx)); err != nil {
		log.Printf("end Pocket ID session: %v", err)
	}
}()
user, err := client.Decode[client.UserDto](c.GetCurrentUser(ctx))
```

`NewSession` returns the generated `*Client` with a private cookie jar and sends `remember=false`. Its final argument accepts a custom `*http.Client` whose settings are copied and transport shared; pass `nil` for defaults. Create the one-time token separately with your admin client. The caller owns logout and cleanup deadlines.

## Errors

`Decode` and `Check` wrap the raw methods and turn every non-2xx response into an `*APIError`, even when the body isn't Pocket ID's error JSON:

```go
user, err := client.Decode[*client.UserDto](c.GetUserByID(ctx, id))
if apiErr := client.AsAPIError(err); apiErr != nil && apiErr.Code == client.CodeNotFound {
	// ...
}
err = client.Check(c.DeleteUser(ctx, id))
```

The `Code` constants are generated from upstream's `apperror` package.

## Updates

OIDC client ID path parameters take `client.OIDCClientID(rawID)`. Pass the raw ID, including the full metadata URL for a CIMD client; the client encodes every ID as `~base64url`, which Pocket ID decodes and validates. Keep JSON body IDs as strings.

PATCH bodies are JSON merge patches: omitted fields are left alone. Set fields with `new(v)`, and clear nullable ones with `nullable.NewNullNullable`:

```go
err = client.Check(c.PatchOIDCClient(ctx, id, client.PatchOIDCClientJSONRequestBody{
	Name:      new("renamed"),
	LaunchURL: nullable.NewNullNullable[string](),
}))
```

## Pagination

`All` yields every item of a list operation, fetching pages as it goes:

```go
for u, err := range client.All[client.UserDto](func(page int) (*http.Response, error) {
	return c.ListUsers(ctx, &client.ListUsersParams{PaginationPage: &page, PaginationLimit: new(100)})
}) {
	if err != nil {
		return err
	}
	// ...
}
```

## Development

Tooling (Go, oapi-codegen, oasdiff, zizmor) is managed with [mise](https://mise.jdx.dev); `jq` must also be on your `PATH`. Run `mise install`, then:

```sh
mise run generate   # clone upstream at POCKET_ID_VERSION, generate openapi.json + client, tidy
mise run verify     # check the generated code compiles
mise run clean      # remove the upstream clone and generated files
```

Generated code is not committed to `main`. Each release tag is a commit containing only `LICENSE`, `README.md`, `go.mod`, `go.sum`, `client/` and `openapi.json`, parented on the previous release so tags can be compared.

## Operation names

Upstream's operationIds are `METHOD_/path`, so `generate-client` names each operation after its summary instead (minus articles, `'s` and the `.0` of version numbers), e.g. "Get API by ID" becomes `GetAPIByID` and "Get OAuth 2.0 …" becomes `GetOAuth2…`. PATCH's "Update…" becomes `Patch…` so it doesn't collide with PUT. Operations without a summary, or whose name would clash, keep upstream's operationId. The published `openapi.json` is unchanged, so if upstream rewords a summary, the method gets renamed.

## Development builds

The `next` branch gets the client generated from upstream Pocket ID `main` whenever it or this repo's `main` changes the output, under the major version upstream is developing:

```sh
go get github.com/aclerici38/pocket-id-go-client/v3@next
```
