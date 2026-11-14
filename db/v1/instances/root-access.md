# Root Access

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
)

func rootEnabled(ctx context.Context, conn *openstack.Connection, instanceID string) (bool, error) {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return false, err
	}
	return service.Instances.IsRootEnabled(ctx, instanceID)
}
```

```python
from openstack.database.v1.instance import Instance

instance = Instance(id=instance_id)
enabled = instance.is_root_enabled(conn.database)
```

## References

- [`Instance.is_root_enabled(session)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/database/v1/instance.py#L73-L85)
- [Trove Root Enabled Contracts Test](../../../api/trove_root_enabled_contracts_test.go)
- [..](../README.md)
