# Aggregate image precaching

`service.Aggregates.PrecacheImages` maps pinned openstacksdk `aggregate_precache_images`. It sends one `POST /os-aggregates/{id}/images` with `{"cache":[{"id":"image"}]}` and **microversion 2.81**, matching Aggregate.precache_images' fixed class version. It performs no image/aggregate lookup and preserves image order, duplicates and empty strings. An empty or nil slice sends `cache:[]`; it does not skip the request.

The action accepts HTTP100–399 and has no JSON result. Empty, opaque and invalid JSON response bodies are accepted. Read/Close/cancellation/source failures remain errors; a successful status cannot hide a physical failure. `WithPrecacheImagesHeader` carries owned headers. An option callback changing the selected API or route fails before sending; the request images are snapshotted before callbacks. HTTP errors keep native status/body/header evidence; accepted physical failures retain `resource.ResponseError`.

Python accepts a scalar image string or list and an Aggregate object/string ID. Go uses an integer aggregate ID and `[]string`; wrap a single image in a one-element slice. Arbitrary Python nonstring image values and cached mutable Resource/session state remain outside this typed Go profile. Source's fixed 2.81 is preserved even when the selected client previously uses another version; the client itself is not mutated. This mapping does not prove real Nova/Glance interoperability or full Resource/session parity.

## Standalone Python

```python
import openstack


def main():
    conn = openstack.connect()
    conn.compute.aggregate_precache_images("7", ["image-one", "image-two"])


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_COMPUTE_ENDPOINT` to the authenticated project-scoped Nova v2.1 endpoint and `OS_TOKEN` to a token authorized for this action.

```go
package main

import (
    "context"
    "log"
    "os"

    compute "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    endpoint := os.Getenv("OS_COMPUTE_ENDPOINT")
    if endpoint == "" { log.Fatal("OS_COMPUTE_ENDPOINT is required") }
    provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
    client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "compute"}
    service := compute.New(client)
    if err := service.Aggregates.PrecacheImages(context.Background(), 7, []string{"image-one", "image-two"}); err != nil {
        log.Fatal(err)
    }
}
```

Focused tests are in `python_precache_images_test.go`; HTTP fixtures reuse the existing Gophercloud testhelper-backed aggregate fixture. No example executes automatically and no live cloud result is claimed.
