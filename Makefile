GOCACHE ?= $(CURDIR)/.go-build

.PHONY: test build check-arch check
test:
	@mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" go test ./...

build:
	@mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" go build ./...

check-arch:
	sh scripts/check_architecture.sh

check: test check-arch
