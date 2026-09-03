release:: generate-client;

.PHONY: generate-client
generate-client:
	go generate ./...
