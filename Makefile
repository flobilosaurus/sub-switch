.PHONY: test vet build static-analysis security-analysis

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./cmd/sub-switch

static-analysis:
	mise run static-analysis

security-analysis:
	mise run security-analysis
