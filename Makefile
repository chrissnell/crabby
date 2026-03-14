VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
IMAGE   := ghcr.io/chrissnell/crabby

.PHONY: all build test vet clean help \
        docker-build docker-push \
        bump-point bump-minor

all: build

help:
	@echo "build        Build crabby binary"
	@echo "test         Run tests"
	@echo "vet          Run go vet"
	@echo "clean        Remove built binary"
	@echo "docker-build Build Docker image"
	@echo "docker-push  Push Docker image"
	@echo "docker       Build and push Docker image"
	@echo "bump-point   Tag and push a patch release"
	@echo "bump-minor   Tag and push a minor release"

build:
	go build $(LDFLAGS) -o bin/crabby ./cmd/crabby

test:
	go test ./cmd/... ./pkg/...

vet:
	go vet ./cmd/... ./pkg/...

clean:
	rm -f bin/crabby

# --- Docker ---

docker-build:
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

docker-push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

docker: docker-build docker-push

# --- Version bumping ---

bump-point:
	@current=$(VERSION); \
	major=$$(echo $$current | sed 's/^v//' | cut -d. -f1); \
	minor=$$(echo $$current | sed 's/^v//' | cut -d. -f2); \
	point=$$(echo $$current | sed 's/^v//' | cut -d. -f3); \
	point=$${point:-0}; \
	new="v$$major.$$minor.$$((point + 1))"; \
	echo "$(VERSION) -> $$new"; \
	git tag -a "$$new" -m "Release $$new"; \
	git push origin "$$new"

bump-minor:
	@current=$(VERSION); \
	major=$$(echo $$current | sed 's/^v//' | cut -d. -f1); \
	minor=$$(echo $$current | sed 's/^v//' | cut -d. -f2); \
	new="v$$major.$$((minor + 1)).0"; \
	echo "$(VERSION) -> $$new"; \
	git tag -a "$$new" -m "Release $$new"; \
	git push origin "$$new"
