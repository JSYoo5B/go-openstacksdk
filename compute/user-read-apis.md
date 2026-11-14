# User Read APIs

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
flavor = conn.compute.get_flavor("flavor-id", get_extra_specs=True)
server = conn.compute.fetch_server_metadata("server-id")
keypair = conn.compute.get_keypair("login-key")
console = conn.compute.get_server_console_output("server-id", length=50)
print(flavor.extra_specs, server.metadata, keypair.fingerprint, console)
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    if len(os.Args) != 4 && len(os.Args) != 5 {
        return fmt.Errorf("usage: %s SERVER_ID FLAVOR_ID KEYPAIR_NAME [OWNER_USER_ID]", os.Args[0])
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    flavor, err := service.Flavors.FindIdentity(ctx, os.Args[2],
        resource.WithIdentityFindFallback(resource.FindFallbackNever),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindExtraSpecs(true),
    )
    if err != nil {
        return fmt.Errorf("get flavor: %w", err)
    }
    metadata, err := service.Servers.Metadata(ctx, os.Args[1])
    if err != nil {
        return fmt.Errorf("get server metadata: %w", err)
    }
    var keyOptions []keypairs.GetOption
    if len(os.Args) == 5 {
        keyOptions = append(keyOptions,
            keypairs.WithGetOptions(keypairs.GetOpts{UserID: os.Args[4]}))
    }
    pair, err := service.KeyPairs.Get(ctx, os.Args[3], keyOptions...)
    if err != nil {
        return fmt.Errorf("get keypair: %w", err)
    }
    output, err := service.Servers.ConsoleOutput(ctx, os.Args[1],
        request.Custom(func(c *request.Config[servers.ConsoleOutputOpts]) error { v := 50; c.Options.Length = &v; return nil }))
    if err != nil {
        return fmt.Errorf("get console output: %w", err)
    }
    data, err := json.MarshalIndent(map[string]any{
        "flavor": flavor,
        "metadata": metadata,
        "keypair": pair,
        "console_output": output,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Finding Identities](../docs/finding-identities.md)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L246-L266)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1876-L1906)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py#L33-L46)
- [get_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L868-L890)
- [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py#L938-L952)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
