build:
	CGO_ENABLED=0 go build -o bin/gptscript -tags "${GO_TAGS}" -ldflags "-s -w" .

test:
	go test -v ./...

validate:
	go vet ./...
	golangci-lint run

ci: build
	./bin/gptscript ./scripts/ci.gpt
