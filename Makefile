VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
IMAGE   := ghcr.io/chrissnell/crabby

.PHONY: all build test vet clean \
        docker-build docker-push \
        bump-point bump-minor

all: build

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
