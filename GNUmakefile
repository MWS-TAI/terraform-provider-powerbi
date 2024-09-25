.PHONY: build, testacc, fmt, fmtcheck, docs, build-all

VERSION ?= 1.0.0

default: build

build: fmtcheck
	@go build

testacc: fmtcheck
	@TF_ACC=1 go test -count=1 -v ./...

fmt:
	@gofmt -l -w $(CURDIR)/internal

fmtcheck:
	@test -z $(shell gofmt -l $(CURDIR)/internal | tee /dev/stderr) || { echo "[ERROR] Fix formatting issues with 'make fmt'"; exit 1; }

docs:
	@go run internal/docgen/cmd/main.go

build-all:
	for GOOS in darwin linux windows; do \
		if [ $$GOOS = "darwin" ]; then \
			GOARCH=arm64; \
		else \
			GOARCH=amd64; \
		fi; \
		if [ $$GOOS = "windows" ]; then \
			EXTENSION=".exe"; \
		else \
			EXTENSION=""; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH"; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -o bin/$(VERSION)/$${GOOS}_$${GOARCH}/terraform-provider-powerbi_v$(VERSION)$$EXTENSION; \
	done