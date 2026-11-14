# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack

def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))

def main():
    conn = openstack.connect()
    identifier = os.environ["IDENTITY_SERVICE_ID"]
    for row in conn.identity.services(
        type="compute", is_enabled=True, max_items=5
    ):
        print_value(row)
    print_value(conn.identity.services(jmespath_filters="[*].id"))
    print_value(conn.search_services(name_or_id="n*", filters="length(@)"))
    print_value(conn.get_service(identifier))
    print_value(conn.get_service(identifier, filters={}))

if __name__ == "__main__":
    main()
```

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printValue(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}
func main() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	identifier := os.Getenv("IDENTITY_SERVICE_ID")
	if identifier == "" {
		log.Fatal("set IDENTITY_SERVICE_ID to a readable service ID or name")
	}
	rows, err := svc.Services.AllRecords(ctx,
		request.Query("type", "compute"),
		request.Query("is_enabled", true),
		request.MaxItems(5),
		request.Custom(func(c *services.RecordListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printValue(row.Resource)
	}
	proxy, err := svc.Services.ProxyListRecords(ctx, services.WithRecordListJMESPath("[*].id"))
	if err != nil {
		log.Fatal(err)
	}
	printValue(proxy.JSON)
	cloud := identity.NewServiceReadCloud(svc.Services)
	found, err := cloud.SearchServices(ctx, "n*",
		identity.WithServiceReadFilters(json.RawMessage(`"length(@)"`)),
		request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(found.Value)
	direct, err := cloud.GetService(ctx, identifier, request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	printValue(direct.Value)
	searched, err := cloud.GetService(ctx, identifier,
		identity.WithServiceReadFilters(json.RawMessage(`{}`)),
		request.Custom(func(c *identity.ServiceReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(searched.Value)
}
```

## References

- [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service.py)
- [Proxy.services/find_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274)
- [Cloud search_services/get_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy.html)
- [Service](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/service.html)
