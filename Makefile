.DEFAULT_GOAL := build

all: build-ui build

build-ui:
	$(MAKE) -C ui
	rm -rf static/ui
	mkdir -p static/ui/_nuxt
	touch static/ui/placeholder static/ui/_nuxt/_placeholder
	cp -rp ui/.output/public/* static/ui/

build:
	CGO_ENABLED=0 go build -o bin/gptscript -tags "${GO_TAGS}" -ldflags "-s -w" .

tidy:
	go mod tidy

test:
	go test -v ./...

GOLANGCI_LINT_VERSION ?= v1.56.1
lint:
	if ! command -v golangci-lint &> /dev/null; then \
  		echo "Could not find golangci-lint, installing version $(GOLANGCI_LINT_VERSION)."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi

	golangci-lint run


validate: tidy lint
	if [ -n "$$(git status --porcelain)" ]; then \
		git status --porcelain; \
		echo "Encountered dirty repo!"; \
		git diff; \
		exit 1 \
	;fi

ci: build
	./bin/gptscript ./scripts/ci.gpt

serve-docs:
	(cd docs && npm start)
