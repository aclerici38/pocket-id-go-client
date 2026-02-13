.PHONY: all clean generate clone-upstream install-tools

POCKET_ID_VERSION := $(shell cat POCKET_ID_VERSION | tr -d '\n')
UPSTREAM_REPO := https://github.com/pocket-id/pocket-id.git
UPSTREAM_DIR := .upstream
SWAGGER_SPEC := swagger.yaml

all: generate

# Install required tools
install-tools:
	go install tool

# Clone upstream at the pinned version
clone-upstream:
	rm -rf $(UPSTREAM_DIR)
	git clone --depth 1 --branch $(POCKET_ID_VERSION) $(UPSTREAM_REPO) $(UPSTREAM_DIR)

# Generate swagger spec from upstream source
generate-spec: clone-upstream
	mkdir -p $(UPSTREAM_DIR)/docs
	touch $(UPSTREAM_DIR)/docs/description.md $(UPSTREAM_DIR)/docs/api.md
	go tool swag init -d $(UPSTREAM_DIR)/backend,$(UPSTREAM_DIR)/backend/internal/controller,$(UPSTREAM_DIR)/backend/internal/dto -g cmd/main.go -ot yaml -o $(UPSTREAM_DIR) -md $(UPSTREAM_DIR)/docs
	cp $(UPSTREAM_DIR)/swagger.yaml $(SWAGGER_SPEC)

# Generate Go client from swagger spec
generate-client:
	go tool swagger generate client -f $(SWAGGER_SPEC) -t . --skip-validation

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
