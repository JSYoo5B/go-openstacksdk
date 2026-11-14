# Memberships

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
user_id = "user-id"

for project in conn.identity.user_projects(user_id):
    print(project.id, project.name, project.domain_id)

for group in conn.identity.user_groups(user_id):
    print(group.id, group.name, group.domain_id)
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	userID := os.Args[1]
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}

	for project, err := range identity.Users.ListProjects(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user projects: %w", err)
		}
		fmt.Printf("project: %s %s domain=%s\n", project.ID, project.Name, project.DomainID)
	}
	for group, err := range identity.Users.ListGroups(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user groups: %w", err)
		}
		fmt.Printf("group: %s %s domain=%s\n", group.ID, group.Name, group.DomainID)
	}
	return nil
}
```

```python
import openstack

conn = openstack.connect()
for project in conn.identity.user_projects(
    "user-id", is_enabled=True, limit=20, max_items=50
):
    print(project.id, project.name, project.domain_id)
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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}
	for record, err := range identity.Users.ListProjectRecords(ctx, os.Args[1],
		users.WithProjectListOptions(users.ListProjectRecordsOpts{Limit: 20, MaxItems: 50}),
		users.WithProjectListFilter("is_enabled", true),
	) {
		if err != nil {
			return fmt.Errorf("list user project records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	for record, err := range identity.Users.ListGroupRecords(ctx, os.Args[1]) {
		if err != nil {
			return fmt.Errorf("list user group records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
```

## References

- [user_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1112-L1131)
- [user_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1414-L1429)
- [UserProject manifest](../../../api/openstacksdk/resources/identity/v3/user_project.json)
