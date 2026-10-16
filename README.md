# pocket-id-go-client

Go client for [Pocket ID](https://github.com/pocket-id/pocket-id), generated from the official OpenAPI (swagger 2.0) spec using [go-swagger](https://github.com/go-swagger/go-swagger).

## Development

Tooling (Go, swag, go-swagger) is managed with [mise](https://mise.jdx.dev). Run `mise install`, then:

```sh
mise run generate   # clone upstream at POCKET_ID_VERSION, generate swagger.yaml + client, tidy
mise run verify     # check the generated code compiles
mise run clean      # remove generated files
```

Generated code is not committed to `main`. Each release tag is a commit containing only `LICENSE`, `go.mod`, `go.sum`, `client/`, `models/` and `swagger.yaml`, parented on the previous release so tags can be compared.
