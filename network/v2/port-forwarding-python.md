# Port Forwarding Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
created = conn.network.create_floating_ip_port_forwarding(
    floating_ip_id, internal_port_id=port_id,
    internal_ip_address="10.0.0.5", internal_port=22,
    external_port=2222, protocol="tcp", description="SSH")
selected = conn.network.get_floating_ip_port_forwarding(
    floating_ip_id, created.id, fields=["id", "external_port"])
for forwarding in conn.network.floating_ip_port_forwardings(
    floating_ip_id, protocol="tcp", description="SSH"):
    print(forwarding.id)
found = conn.network.find_floating_ip_port_forwarding(floating_ip_id, created.id)
#     floating_ip_id, created.id, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
)

func printForwarding(value *portforwarding.PortForwardingRecord) {
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("parent=%s status=%d\n%s\n", value.ParentID, value.StatusCode, body)
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    floatingIP := flag.String("floating-ip", "", "floating IP ID")
    id := flag.String("id", "", "port forwarding ID to get and find")
    port := flag.String("port", "", "internal Neutron port ID")
    address := flag.String("address", "10.0.0.5", "internal IP address")
    create := flag.Bool("create", false, "create a port forwarding")
    description := flag.String("description", "", "replace description, including empty")
    remove := flag.String("delete", "", "port forwarding ID to delete")
    flag.Parse()
    if *floatingIP == "" { log.Fatal("-floating-ip is required") }
    update := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "description" { update = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.PortForwarding
    for row, err := range api.ListRecords(ctx, *floatingIP,
        portforwarding.WithRecordListQuery("protocol", "tcp"),
        request.MaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *id != "" {
        row, err := api.GetRecord(ctx, *floatingIP, *id,
            portforwarding.WithRecordFields("id", "external_port"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
        found, err := api.FindRecord(ctx, *floatingIP, *id)
        if err != nil { log.Fatal(err) }
        printForwarding(found)
        if update {
            row, err := api.UpdateRecord(ctx, *floatingIP, *id,
                portforwarding.WithRecordAttribute("description", *description))
            if err != nil { log.Fatal(err) }
            printForwarding(row)
        }
    }
    if *create {
        row, err := api.CreateRecord(ctx, *floatingIP,
            portforwarding.WithRecordAttribute("internal_port_id", *port),
            portforwarding.WithRecordAttribute("internal_ip_address", *address),
            portforwarding.WithRecordAttribute("internal_port", 22),
            portforwarding.WithRecordAttribute("external_port", 2222),
            portforwarding.WithRecordAttribute("protocol", "tcp"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *floatingIP, *remove,
            request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Reference](README.md)
