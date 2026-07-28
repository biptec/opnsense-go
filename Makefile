modules := $(notdir $(basename $(wildcard schema/*.yml)))
generator := $(shell find internal/generate)

go := go

PKG ?=
TEST ?=

.PHONY: all test testacc fmt fmt-check vet staticcheck check

test:
	$(go) test ./...

testacc:
ifdef PKG
	$(go) test -v -p 1 -timeout 120m $(if $(TEST),-run $(TEST)) ./pkg/$(PKG)/...
else
	$(go) test -v -p 1 -timeout 120m $(if $(TEST),-run $(TEST)) ./pkg/...
endif

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || \
		(echo "Go files need formatting:"; gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); exit 1)

vet:
	$(go) vet ./...

staticcheck:
	$(go) run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...

check: fmt-check test vet staticcheck

pkg/opnsense/client.go: $(generator) $(wildcard schema/*.yml)
	@echo "Generating opnsense client"
	$(go) generate ./pkg/opnsense

pkg/%/controller.go: schema/%.yml $(generator) pkg/%/generate.go
	@echo "Generating $* controller"
	$(go) generate ./$(@D)

$(modules): %: pkg/%/controller.go

all: $(modules) pkg/opnsense/client.go
