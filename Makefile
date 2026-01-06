.PHONY: all clean generate clone-upstream install-tools

POCKET_ID_VERSION := $(shell cat POCKET_ID_VERSION | tr -d '\n')
UPSTREAM_REPO := https://github.com/pocket-id/pocket-id.git
UPSTREAM_DIR := .upstream
SWAGGER_SPEC := swagger.yaml

all: generate

# Install required tools
install-tools:
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# Clone upstream at the pinned version
clone-upstream:
	rm -rf $(UPSTREAM_DIR)
	git clone --depth 1 --branch $(POCKET_ID_VERSION) $(UPSTREAM_REPO) $(UPSTREAM_DIR)

# Generate swagger spec from upstream source
generate-spec: clone-upstream
	mkdir -p $(UPSTREAM_DIR)/docs
	touch $(UPSTREAM_DIR)/docs/description.md $(UPSTREAM_DIR)/docs/api.md
	cd $(UPSTREAM_DIR)/backend && swag init -d .,./internal/controller,./internal/dto -g cmd/main.go -ot yaml -o ../ -md ../docs
	cp $(UPSTREAM_DIR)/swagger.yaml $(SWAGGER_SPEC)

# Generate Go client from swagger spec
generate-client:
	swagger generate client -f $(SWAGGER_SPEC) -t . --skip-validation

# Full generation pipeline
generate: generate-spec generate-client
	go mod tidy
	rm -rf $(UPSTREAM_DIR)

# Clean generated files
clean:
	rm -rf $(UPSTREAM_DIR) client models $(SWAGGER_SPEC)

# Verify the generated code compiles
verify:
	go build ./...
