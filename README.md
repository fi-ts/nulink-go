# Nulink Go

This project provides go types for the nulink API via their provieded openapi json definition.

## Generation

We use https://github.com/oapi-codegen/oapi-codegen for generating a golang client from the spec.

Use `make generate-client` to regenerate.

## Updating to a new nulink version

1. Overwrite [`spec/openapi.json`](spec/openapi.json) with the new OpenAPI spec.
2. Overwrite [`spec/validations.json`](spec/validations.json) with the new validations.
3. Run `make generate-client`.

The nulink version is read from `info.version` in the spec. A test
(`spec/versions_test.go`) fails if it differs from `api_version` in the
validations file.

## Fake server

[`test`](test/fakes.go) ships an in-process fake of the API for tests and local
development:

```go
client, url, cleanup, err := test.Start()
defer cleanup()
```

It serves canned data (currently the VM endpoints, e.g. `GET /api/v1/mvm/os`)
through a real HTTP server, so the generated client exercises the full transport,
