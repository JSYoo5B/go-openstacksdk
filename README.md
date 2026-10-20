# go-openstacksdk

An independent high-level Go SDK for OpenStack, built on Gophercloud.

Connection, service facades, resource collections, and workflows provide consistent authentication, explicit name/ID references, pagination, waits, concrete request options, and preserved partial results. The SDK is under development; native API bindings and Python openstacksdk behavior coverage are tracked separately.

Requires Go 1.25 or later and Gophercloud v2.15.0. The module is `github.com/JSYoo5B/go-openstacksdk`. This project is independent of OpenStack, the OpenInfra Foundation, and Gophercloud; no affiliation, endorsement, or sponsorship is implied.

## Quick start

```sh
go get github.com/JSYoo5B/go-openstacksdk@latest
```

Configure authentication using standard OpenStack environment variables or a selected clouds.yaml cloud. The [installation guide](docs/install.md) and [connection configuration](connection_options.go) describe the available options.

```go
package main

import (
    "context"
    "fmt"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { panic(err) }
    service, err := conn.Compute(ctx)
    if err != nil { panic(err) }
    server, err := service.Servers.Find(ctx, resource.Name("example-server"))
    if err != nil { panic(err) }
    fmt.Println(server.ID, server.Status)
}
```

This example performs authenticated reads. [Create-server](examples/create-server/main.go) demonstrates explicit mutation inputs and partial results when a wait fails.

## Features and boundaries

- Shared provider authentication, service caching, endpoint selection, TLS, and explicit or negotiated microversions.
- Generated Gophercloud facades with concrete options and SDK-owned request builders; SDK-owned APIs extend selected services absent from the pinned dependency.
- Resource collections with exact name matching, duplicate detection, missing-resource policies, context cancellation, lazy pagination, and resource-specific waits.
- Parent scopes for child resources, quotas, metadata, and compound identities.
- Selected compute/network/storage/image workflows with ordered requests and partial-success evidence.
- Owned record APIs preserve declared views, actual wire values, raw bodies, headers, and status according to each operation's contract.

The SDK does not claim complete Python Resource, descriptor, session, cache, or cloud-helper equivalence. Supported inputs, accepted statuses, encoding profiles, ownership, and microversion behavior vary by operation. Mock HTTP and source-comparison tests do not establish authenticated cloud acceptance. Consult the [support ledger](docs/sdk-support-ledger.md), [operation reviews](api/sdk_reviews.json), and [current coverage](docs/implementation-plan.md).

## Documentation

- [API reference](api/README.md) and [Python source inventory](api/openstacksdk/README.md)
- [Design](docs/design.md), [identity lookup](docs/finding-identities.md), [listing](docs/listing.md), [parent scopes](docs/scoped-resources.md), [waits](docs/service-waits.md), and [microversions](docs/microversions.md)
- [Installation](docs/install.md), [testing](docs/testing.md), [release criteria](docs/release-milestones.md), and [service priorities](docs/implementation-plan.md)

## Development

```sh
make check
make smoke
make generate
```

`make check` covers licensing, vet, race tests, support-reference consistency, generated progress, and formatting. `make smoke` runs bounded HTTP fixture workflows. Generation reads pinned upstream source and may access the network. See [testing](docs/testing.md) for source-backed generator checks and reproducibility.

## License

Project code is licensed under [Apache-2.0](LICENSE). Preserve [NOTICE](NOTICE), [third-party notices](THIRD_PARTY_NOTICES.md), and the [license texts](licenses/README.md). See [licensing](docs/licensing.md) for component attribution and contribution rules.
