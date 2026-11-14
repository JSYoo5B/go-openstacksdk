# User Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_server_remote_console("server-id", type="novnc")
print(console.protocol, console.type, console.url)
conn.compute.delete_keypair("old-login-key")
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles"
)

func main() {
    serverID := flag.String("server-id", "", "server ID for the remote console")
    keyName := flag.String("keypair", "", "keypair name to delete")
    ownerID := flag.String("owner-user-id", "", "optional keypair owner")
    strict := flag.Bool("strict-missing", false, "report a missing keypair")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *keyName, *ownerID, *strict); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, keyName, ownerID string, strict bool) error {
    if serverID == "" || keyName == "" {
        return fmt.Errorf("-server-id and -keypair are required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    console, err := service.RemoteConsoles.CreateConsole(ctx, serverID,
        request.Custom(func(c *request.Config[remoteconsoles.ConsoleCreateOpts]) error { c.Options.Type = request.Present("novnc"); return nil }))
    if console != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "resource": console.Resource,
            "wire": console.Wire,
            "status_code": console.StatusCode,
            "header": console.Header,
        }, "", "  ")
        if printErr != nil {
            return printErr
        }
        fmt.Println(string(data))
    }
    if err != nil {
        return fmt.Errorf("create remote console: %w", err)
    }
    options := []keypairs.KeypairDeleteOption{
        request.IgnoreMissing(!strict),
    }
    if ownerID != "" {
        options = append(options, request.Custom(func(c *request.Config[keypairs.KeypairDeleteOpts]) error { c.Options.UserID = ownerID; return nil }))
    }
    if err := service.KeyPairs.DeleteKeypair(ctx, keyName, options...); err != nil {
        return fmt.Errorf("delete keypair: %w", err)
    }
    fmt.Println("keypair deletion accepted or missing")
    return nil
}
```

## References

- [Console Selection](console-selection.md)
- [create_server_remote_console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2903-L2920)
- [Server Remote Console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_remote_console.py#L18-L77)
- [delete_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L842-L866)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1404)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
