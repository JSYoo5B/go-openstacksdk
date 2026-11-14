# Python Group Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--group-id", required=True)
    args = parser.parse_args()
    conn = openstack.connect()
    for row in conn.identity.group_users(
        args.group_id, email="member@example.com", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    groupID := flag.String("group-id", "", "required group UUID")
    flag.Parse()
    if *groupID == "" { log.Fatal("-group-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Users.ListUsersInGroupRecords(ctx, *groupID,
        users.WithGroupRecordListQuery("email", "member@example.com"),
        request.Limit(100),
        request.MaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## References

- [Group](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/groups.py)
- [Python Records](python_records.md)
