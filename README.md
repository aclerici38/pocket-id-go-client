# pocket-id-go-client

Go client for [Pocket ID](https://github.com/pocket-id/pocket-id), generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) from the OpenAPI 3.1 spec that upstream builds from its route definitions.

```sh
go get github.com/aclerici38/pocket-id-go-client/v2
```

Release tags match Pocket ID versions, and the module's major version follows Pocket ID's.

## Development

Tooling (Go, oapi-codegen, oasdiff, zizmor) is managed with [mise](https://mise.jdx.dev); `jq` must also be on your `PATH`. Run `mise install`, then:

```sh
mise run generate   # clone upstream at POCKET_ID_VERSION, generate openapi.json + client, tidy
mise run verify     # check the generated code compiles
mise run clean      # remove the upstream clone and generated files
```

Generated code is not committed to `main`. Each release tag is a commit containing only `LICENSE`, `go.mod`, `go.sum`, `client/` and `openapi.json`, parented on the previous release so tags can be compared.

## Operation names

Upstream's operationIds are `METHOD_/path`, so `generate-client` names each operation after its summary instead (minus articles, `'s` and the `.0` of version numbers), e.g. "Get API by ID" becomes `GetAPIByID` and "Get OAuth 2.0 …" becomes `GetOAuth2…`. PATCH's "Update…" becomes `Patch…` so it doesn't collide with PUT. Operations without a summary, or whose name would clash, keep upstream's operationId. The published `openapi.json` is unchanged, so if upstream rewords a summary, the method gets renamed.

## Development builds

The `next` branch gets the client generated from upstream Pocket ID `main` whenever it or this repo's `main` changes the output, under the major version upstream is developing:

```sh
go get github.com/aclerici38/pocket-id-go-client/v3@next
```
