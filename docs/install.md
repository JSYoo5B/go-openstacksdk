# Installation and external use

Requires Go 1.25 or later. The SDK is under development; consult [coverage](implementation-plan.md) and the [support ledger](sdk-support-ledger.md) before selecting an API. No release tag is implied by this guide.

```sh
go mod init example.com/mycloud
GOWORK=off go get github.com/JSYoo5B/go-openstacksdk@latest
```

For reproducible builds, replace `latest` with a published version or an explicitly selected commit. `go list -m -json github.com/JSYoo5B/go-openstacksdk` shows the resolved version. Do not add filesystem replace directives to a published consumer module.

The [quick start](../README.md) demonstrates authentication and a read. [Create-server](../examples/create-server/main.go) compiles as a complete application and performs real mutations only when explicitly run with its required inputs. Authentication supports standard OpenStack environment variables and cloud configuration; see [Connection options](../connection_options.go).

## Consumer API compilation

The following program references public APIs without authenticating or executing HTTP requests. It demonstrates root, service, leaf, concrete-option, and generic-option imports.

```go
package main

import (
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    _ = sdk.Connect
    _ = compute.CreateServerRequest{}
    _ = resource.ID("example-id")
    _ = request.Present(false)
}
```

Build an external module with `GOWORK=off go build ./...`. Module download/build success proves importability for that version; it does not establish authenticated cloud interoperability.
