# Cloud User Role Get Python

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
    user = os.environ["IDENTITY_USER_REF"]
    role = os.environ["IDENTITY_ROLE_REF"]
    domain = os.getenv("IDENTITY_DOMAIN_ID")
    print_value(conn.get_user(user, domain_id=domain))
    print_value(conn.get_user(user, filters={}, domain_id=domain))
    # Pinned Source checks len([None]) then still calls find_user.
    print_value(conn.get_user(user, filters="`[null]`", domain_id=domain))
    print_value(conn.get_role(role, domain_id=domain))
    # Role returns entities[0], so this expression returns None.
    print_value(conn.get_role(role, filters="`[null]`", domain_id=domain))

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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
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
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	user, role := os.Getenv("IDENTITY_USER_REF"), os.Getenv("IDENTITY_ROLE_REF")
	if user == "" || role == "" {
		log.Fatal("set IDENTITY_USER_REF and IDENTITY_ROLE_REF to readable IDs or names")
	}
	userOpts := []identity.UserGetOption{}
	roleOpts := []identity.RoleGetOption{}
	if domain, ok := os.LookupEnv("IDENTITY_DOMAIN_ID"); ok {
		userOpts = append(userOpts, identity.WithUserGetDomainID(domain))
		roleOpts = append(roleOpts, identity.WithRoleGetDomainID(domain))
	}
	directUser, err := conn.GetUserRecord(ctx, user, userOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directUser.Value)
	filteredUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetFilters(json.RawMessage(`{}`)))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredUser.Value)
	pinnedUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(pinnedUser.Value)
	directRole, err := conn.GetRoleRecord(ctx, role, roleOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directRole.Value)
	filteredRole, err := conn.GetRoleRecord(ctx, role, append(roleOpts,
		identity.WithRoleGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredRole.Value)
}
```

## References

- [get_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344)
- [get_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422)
- [Proxy find_user/find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [User](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/user.html)
- [Role](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/role.html)
- [User](v3/users/python_records.md)
- [Role](v3/roles/python-records.md)
