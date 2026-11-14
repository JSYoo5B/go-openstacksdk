# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack

def main():
    conn = openstack.connect()
    domain = os.environ["USER_QUERY_DOMAIN_ID"]
    for row in conn.identity.users(
        domain_id=domain, is_enabled=True, max_items=10
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    listed = conn.list_users(domain_id=domain)
    print(json.dumps([row.to_dict() for row in listed], indent=2))
    searched = conn.search_users(
        name_or_id="team*", domain_id=domain, filters="length(@)"
    )
    print(json.dumps(searched, indent=2))

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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
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
	domain := os.Getenv("USER_QUERY_DOMAIN_ID")
	if domain == "" {
		log.Fatal("set USER_QUERY_DOMAIN_ID to an allowed domain ID")
	}

	rows, err := svc.Users.AllRecords(ctx,
		users.WithRecordListQuery("domain_id", domain),
		users.WithRecordListQuery("is_enabled", true),
		request.MaxItems(10),
		request.Custom(func(c *users.RecordListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printJSON(row.Resource.Body)
	}

	cloud := identity.NewUserReadCloud(svc.Users)
	listed, err := cloud.ListUsers(ctx,
		identity.WithUserListQuery("domain_id", domain),
		request.Custom(func(c *identity.UserListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(listed.Value)

	searched, err := cloud.SearchUsers(ctx, "team*",
		request.Custom(func(c *identity.UserSearchOpts) error { c.DomainID = domain; return nil }),
		identity.WithUserSearchFilters(json.RawMessage(`"length(@)"`)),
		request.Custom(func(c *identity.UserSearchOpts) error { loc := location.Clone(); c.Location = &loc; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(searched.Value)
}
```

## References

- [User](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/user.py)
- [Proxy.users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429)
- [Cloud list_users/search_users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L290)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy)
- [Python Record Find](python-record-find.md)
