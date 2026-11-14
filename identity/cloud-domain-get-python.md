# Cloud Domain Get Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--domain-id", required=True)
    parser.add_argument("--domain-name", default="")
    args = parser.parse_args()
    conn = openstack.connect()
    row = conn.identity.get_domain(args.domain_id)
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    row = conn.get_domain(domain_id=args.domain_id,
                          filters={"ignored": "without evaluation"})
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    if args.domain_name:
        found = conn.get_domain(name_or_id=args.domain_name)
        if found is not None:
            print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    "github.com/JSYoo5B/go-openstacksdk/request"
    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    id := flag.String("domain-id", "", "required domain UUID")
    name := flag.String("domain-name", "", "optional domain name")
    flag.Parse()
    if *id == "" { log.Fatal("-domain-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Domains.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource.Body)

    cloud := identity.NewDomainReadCloud(service.Domains)
    result, err := cloud.GetDomain(ctx,
        request.Custom(func(c *identity.DomainGetOpts) error { v := *id; c.DomainID = &v; return nil }),
        identity.WithDomainGetFilters(json.RawMessage(`{invalid-but-ignored`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(result.Record.Resource.Body)
    fmt.Println("deprecated filters ignored:", result.DeprecatedFilters)
    if *name != "" {
        found, err := cloud.GetDomain(ctx, request.Custom(func(c *identity.DomainGetOpts) error { v := *name; c.NameOrID = &v; return nil }))
        if err != nil { log.Fatal(err) }
        if found.Record != nil { printJSON(found.Record.Resource.Body) }
    }
}
```

## References

- [Python Records](v3/domains/python-records.md)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [list_domains](v3/domains/python-records.md)
