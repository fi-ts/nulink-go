# Nulink Go

This project provides go types for the nulink API via their provieded openapi json definition.

## Generation

We use https://github.com/oapi-codegen/oapi-codegen for generating a golang client from the spec.

Use `make generate-client` to regenerate.

## Updating to a new nulink version

Provide the server credentials as environment variables (`NULINK_SERVER_URL` with a trailing slash, `NULINK_USER`, `NULINK_PW`) and let make do the rest:

```sh
make update-spec
```

`make update-spec` fetches the OpenAPI spec and the validations, regenerates
the client and runs the version check. The steps are also available alone:

- `make fetch-spec` writes [`spec/openapi.json`](spec/openapi.json)
- `make fetch-validations` writes [`spec/validations.json`](spec/validations.json)
- `make generate-client` regenerates the Go code

Existing files are only replaced if the request succeeds. Requires `curl` and `jq`.

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
