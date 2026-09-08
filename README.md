# Nulink Go

This project provides go types for the nulink API via their provieded openapi json definition.

## Generation

We use https://github.com/oapi-codegen/oapi-codegen for generating a golang client from the spec.

Use `make generate-client` to regenerate.

## Fake server

[`fake`](fake/fake.go) ships an in-process fake of the API for tests and local
development:

```go
client, url, cleanup, err := fake.Start()
defer cleanup()
```

It serves canned data (currently the VM endpoints, e.g. `POST /api/v1/vm/os/list`)
through a real HTTP server, so the generated client exercises the full transport,
