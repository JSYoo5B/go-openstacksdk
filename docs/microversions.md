# Microversion discovery and selection

The SDK can discover a cloud's advertised microversion range and select the
highest version shared with your application. This is an opt-in connection
policy. A connection with no microversion option keeps the existing behavior:
no discovery request and no microversion header.

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx,
        sdk.WithMicroversionRange(sdk.Compute, "2.52", "2.90"),
        sdk.WithLatestMicroversion(sdk.BlockStorage),
    )
    if err != nil { panic(err) }

    compute, err := conn.ComputeV2(ctx)
    if err != nil { panic(err) }
    selection, err := conn.Microversion(ctx, sdk.Compute)
    if err != nil { panic(err) }
    fmt.Println(selection.RequestedMinimum, selection.RequestedMaximum)
    fmt.Println(selection.SupportedMinimum, selection.SupportedMaximum)
    fmt.Println(selection.Selected, compute.RawClient().Microversion)
}
```

`WithMicroversionRange(service, minimum, maximum)` has inclusive bounds. Empty
minimum uses the cloud's minimum. Empty maximum or `"latest"` uses the cloud's
maximum. `WithLatestMicroversion(service)` is the unbounded form. The SDK compares
major and minor numbers, so `2.100` is greater than `2.90`. The highest version in
the intersection is selected. An invalid or reversed range fails when constructing
the connection. A valid range with no cloud overlap fails when accessing the service
with an error wrapping `resource.ErrUnsupported`.

An upper bound lets an application preserve its expected response contract when a
cloud adds new microversions. An unbounded latest policy will select a newer version
when a new connection discovers an upgraded cloud.

## Exact version precedence

`WithMicroversion(service, version)` pins an exact version and bypasses discovery.
It takes precedence over automatic range/latest selection regardless of option
order, including when that exact version lies outside an otherwise configured
range. The range still must be syntactically valid. Repeated options of the same
kind use the last value.

```go
conn, err := sdk.Connect(ctx,
    sdk.WithLatestMicroversion(sdk.Compute),
    sdk.WithMicroversion(sdk.Compute, "2.52"),
)
```

The exact pin preserves the previous SDK contract: the cloud checks whether that
version is usable when an API request is made. `MicroversionSelection` reports the
exact request and any configured range separately. `Selected` is the value sent on
subsequent requests. `Negotiated`, `SupportedMinimum`, `SupportedMaximum`, and
`DiscoveryURL` describe actual discovery; those fields remain false/empty for an
exact pin or the default policy.

## Discovery and cache behavior

Discovery runs lazily during the first service access, using that access's context
and the connection's authenticated provider, HTTP transport, region, and endpoint
interface. Connection creation itself does not contact service discovery endpoints.
`Connection.Microversion(ctx, service)` also lazily initializes the service client,
then returns a copy of its selection information.

For Nova, Cinder, Manila, and Masakari project-scoped catalog URLs, the SDK removes the project
segment for discovery: `/compute/v2.1/project/` is inspected at `/compute/v2.1/`,
then `/compute/` when the version endpoint returns 404/405 or omits advertised bounds.
Reverse-proxy prefixes remain intact. Cinder `/volume/v3/project/` is inspected at
`/volume/v3/`. Versioned Ironic and Magnum endpoints likewise have a root fallback.
Unversioned discovery endpoints such as Placement's root are used directly.
Masakari `/instance-ha/v1/project/` is inspected at `/instance-ha/v1/` and
then `/instance-ha/`. Senlin's unversioned catalog root is extended with `/v1`
for resource requests; discovery can fall back from that version to the root.

The decoder accepts a `version` object, a `versions` array, a `versions.values`
array, and an unenveloped version object, including `version` or `max_version`
maximum fields. It matches the API version already selected by the client and
ignores other major APIs and experimental versions. An explicit legacy Nova `v2`
endpoint is not upgraded to `v2.1`; supply a `v2.1` endpoint to negotiate Nova
microversions. Discovery does not replace the API endpoint or follow advertised
links to other hosts.

Malformed documents and HTTP/authentication failures are returned with their
underlying errors intact. Only 404/405 and a valid document without usable bounds
trigger the root fallback. Failed discovery or negotiation does not enter the
client cache and can be retried. Concurrent first accesses share one successfully
configured client and one discovery result. The high-level service and its
versioned `.API` service share that client and the selected request headers.
Cached clients retain their negotiated version for the connection's lifetime;
create a new connection to discover an updated range. Treat `RawClient()` settings
as immutable while sharing a connection across goroutines.

## Services and Python comparison

Automatic selection is limited to services whose API exposes
microversion semantics: Compute v2, Block Storage v3, Bare Metal v1, Bare Metal
Introspection v1, Container v1, Container Infrastructure v1, Placement v1, and
Shared File System v2, and the SDK-owned Accelerator v2, Clustering v1 and
Instance HA v1 services. It requires the cloud to advertise valid bounds. Network,
Image, Identity, Object Storage, and other services without those semantics reject
microversion options with `resource.ErrUnsupported`. A Block Storage microversion
policy applies to v3 and cannot silently apply to `BlockStorageV2()`.

In the pinned Python SDK, service discovery and default microversions are configured
through `CloudRegion` and keystoneauth adapters. An API version containing a minor
component can imply a default microversion, and configured defaults are checked
against discovered bounds. A Python exact request such as
`openstack.connect(compute_api_version="2.52")` maps to
`WithMicroversion(Compute, "2.52")` as a request policy. The Go SDK preserves its
existing exact-pin behavior and makes automatic cloud intersection a separate,
explicit option; it does not claim identical Python discovery defaults.

The HTTP tests in [microversion_test.go](../microversion_test.go) verify catalog and
proxy URL shapes, authentication, numeric ordering, request headers, exact pin
precedence, shared concurrent clients, cancellation, retries, unsupported services,
invalid ranges, non-overlap, malformed responses, and root fallbacks.
