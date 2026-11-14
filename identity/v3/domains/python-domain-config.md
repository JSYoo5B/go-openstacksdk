# Python Domain Config

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--domain", required=True)
parser.add_argument("--action", choices=("create", "get", "update", "delete"), required=True)
parser.add_argument("--attrs", default='{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}')
parser.add_argument("--strict-delete", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.action == "create":
        value = conn.identity.create_domain_config(args.domain, **json.loads(args.attrs))
    elif args.action == "get":
        value = conn.identity.get_domain_config(args.domain)
    elif args.action == "update":
        value = conn.identity.update_domain_config(args.domain, **json.loads(args.attrs))
    else:
        value = conn.identity.delete_domain_config(args.domain, ignore_missing=not args.strict_delete)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    domain := flag.String("domain", "", "parent Domain ID")
    action := flag.String("action", "", "create, get, update or delete")
    attrs := flag.String("attrs", `{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}`, "create/update JSON attributes")
    strict := flag.Bool("strict-delete", false, "do not ignore missing config")
    flag.Parse()
    if *domain == "" { log.Fatal("domain is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    var row *domains.DomainConfigRecord
    switch *action {
    case "create", "update":
        var fields map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*attrs), &fields); err != nil { log.Fatal(err) }
        if fields == nil { log.Fatal("attrs must be a JSON object") }
        option := domains.WithDomainConfigOpts(domains.DomainConfigOpts{Attributes: fields})
        if *action == "create" {
            row, err = service.Domains.CreateConfigRecord(ctx, *domain, option)
        } else {
            row, err = service.Domains.UpdateConfigRecord(ctx, *domain, option)
        }
    case "get":
        row, err = service.Domains.GetConfigRecord(ctx, *domain)
    case "delete":
        result, err := service.Domains.DeleteConfigRecord(ctx, *domain, request.Custom(func(c *domains.DomainConfigOpts) error { v := !*strict; c.IgnoreMissing = &v; return nil }))
        if err != nil { log.Fatal(err) }
        printJSON(result.Deleted)
        return
    default:
        log.Fatal("action must be create, get, update or delete")
    }
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource)
}
```

## References

- [Domain scalar-ID update](python-record-mutations.md)
- [Install](../../../docs/install.md)
