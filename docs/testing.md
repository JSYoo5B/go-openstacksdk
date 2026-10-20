# Testing

Run commands at the repository root with a supported Go toolchain. Go 1.27.2 is the formatting reference; the module's minimum language/toolchain requirement remains Go 1.25.

```sh
make check
make smoke
go build ./...
```

`make check` runs the license check, `go vet ./...`, `go test -race -timeout 60s ./...`, support-reference validation, generated progress validation, and `gofmt` cleanliness. Use the reference toolchain's `gofmt` as well as its `go` command. `make smoke` selects nine test groups covering server/network, volume, image, secret, and Swift object workflows. It fails when a required group is missing or skipped, and writes `.reports/core-smoke.json` and `.reports/core-smoke.log`.

## Source-backed generator validation

`make generate` downloads pinned Gophercloud metadata and reads a pinned Python openstacksdk checkout as AST data. `SDKGEN_OPENSTACKSDK_SOURCE` can select an existing related-source checkout; the generator verifies the pinned source. The filter-manifest extractor requires Python 3.14 because its recorded AST hashes depend on the parser minor version. Python is not a product runtime dependency.

Generator tests use `OPENSTACKSDK_SOURCE` for an explicitly supplied upstream checkout and `GOPHERCLOUD_METADATA` for compiled Gophercloud metadata. Without these inputs, the corresponding source-backed portions skip; fixture-based tests still run. Set both variables for a complete source-backed generator run. No fixed checkout or metadata location is assumed.

```sh
make generate
make progress
make check
```

Generated code and documentation must remain reproducible. Test generation against the same source pins, compare outputs, and review changes rather than inferring parity from successful generation.

## Verification scope

Tests use Gophercloud test helpers and disposable HTTP servers to verify paths, encoded queries, request bodies, headers, accepted statuses, response ownership, exact numbers, nullable fields, iteration, context cancellation, and partial failures. Negative cases check protected fields, unsafe continuations, duplicate identities, malformed responses, errors after acceptance, and mutation replay boundaries.

Pinned-source AST comparisons and bounded Python references verify only their declared scope. Local HTTP tests do not exercise real credentials, deployment policy overrides, service databases, or asynchronous daemon completion. An import acknowledgement is separate from an ACTIVE image; a deletion acknowledgement is separate from observed absence. Live cloud acceptance requires a disposable authorized deployment and explicit role/version/configuration evidence.

Development checks run locally with HTTP fixtures. Live integration requires an authorized disposable deployment. Keep generated reports untracked. Examples that mutate resources require deliberately supplied cloud configuration and inputs.
