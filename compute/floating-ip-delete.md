# Floating Ip Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
result, err := conn.DeleteFloatingIP(ctx,
	compute.DeleteFloatingIPRequest{ID: ipID},
	func(c *compute.FloatingIPDeleteOpts) error { c.Retries = 1; return nil },
	compute.WithFloatingIPDeleteTimeout(time.Minute),
)
```

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	id := flag.String("id", "", "floating IP ID to delete")
	retries := flag.Int("retry", 1, "additional deletes; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	direct := flag.Bool("direct", false, "use direct GET for a UUID verification ID")
	flag.Parse()
	if *id == "" {
		return errors.New("-id is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
	)
	if err != nil {
		return err
	}
	opts := []compute.FloatingIPDeleteOption{
		func(c *compute.FloatingIPDeleteOpts) error { c.Retries = *retries; return nil },
		func(c *compute.FloatingIPDeleteOpts) error { c.DirectGet = *direct; return nil },
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	}
	if *source != "" {
		opts = append(opts, func(c *compute.FloatingIPDeleteOpts) error { v := compute.FloatingIPSource(*source); c.Source = &v; return nil })
	}
	result, deleteErr := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: *id}, opts...)
	var summary map[string]any
	if result != nil {
		accepted := 0
		for _, attempt := range result.Attempts {
			if attempt.Accepted {
				accepted++
			}
		}
		summary = map[string]any{
			"id": result.ID, "backend": result.Backend,
			"deleted": result.Deleted, "absent": result.Absent, "down": result.Down,
			"attempts": len(result.Attempts), "accepted_attempts": accepted,
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(deleteErr, encoder.Encode(summary))
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_floating_ip("IP_ID")
accepted = conn.delete_floating_ip("IP_ID", retry=0)
```

## References

- [Floating Ip Queries](floating-ip-queries.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [public retry/verification](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1017)
- [Neutron/Nova delete helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1019-L1046)
- [ Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L148-L201)
- [Available](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
