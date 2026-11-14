# Metadata Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    kind := flag.String("kind", "container", "container, order or secret")
    id := flag.String("id", "", "explicit resource ID")
    strict := flag.Bool("strict", false, "return an error when the ID is missing")
    timeout := flag.Duration("timeout", time.Minute, "authentication and deletion timeout")
    flag.Parse()
    if err := run(*cloud, *kind, *id, *strict, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, kind, id string, strict bool, timeout time.Duration) error {
    if timeout <= 0 || (kind != "container" && kind != "order" && kind != "secret") {
        return fmt.Errorf("positive timeout and kind container/order/secret are required")
    }
    ref := resource.ID(id)
    if err := ref.Validate(); err != nil {
        return err
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    var options []resource.LookupOption
    if strict {
        options = append(options, resource.WithMissingError())
    }
    switch kind {
    case "container":
        err = service.Containers.Remove(ctx, ref, options...)
    case "order":
        err = service.Orders.Remove(ctx, ref, options...)
    case "secret":
        err = service.Secrets.Remove(ctx, ref, options...)
    }
    if err != nil {
        return err
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(map[string]any{
        "kind": kind,
        "request_id": id,
        "ignore_missing": !strict,
        "completed": true,
    })
}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L58)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L169)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L270)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
