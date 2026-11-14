# Cloud Firewall Deletes Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

The filters JSON may carry Source Find kwargs such as `{"headers": {"X-Trace": "t"}, "microversion": "2.5", "max_items": 1}`: headers become an initial GET query value and fallback LIST headers, microversion versions only the initial GET, and max_items is compared before each fallback row.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("rule", "policy", "group"), default="rule")
parser.add_argument("--name", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "rule":
        result = conn.delete_firewall_rule(args.name, filters=filters)
    elif args.kind == "policy":
        result = conn.delete_firewall_policy(args.name, filters=filters)
    else:
        result = conn.delete_firewall_group(args.name, filters=filters)
    print(json.dumps(result))
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
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "rule", "rule, policy or group")
	name := flag.String("name", "", "literal resource ID or exact name to delete")
	filters := flag.String("filters", "{}", "JSON Source Find kwargs")
	flag.Parse()
	if *name == "" {
		log.Fatal("name is required")
	}
	if *kind != "rule" && *kind != "policy" && *kind != "group" {
		log.Fatal("unknown kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	opts := []network.FirewallDeleteOption{network.WithFirewallDeleteFilters(json.RawMessage(*filters)), request.Custom(func(c *network.FirewallDeleteOpts) error { loc := location.Clone(); c.CurrentLocation = &loc; return nil })}
	var result *network.FirewallDeletion
	switch *kind {
	case "rule":
		result, err = service.DeleteFirewallRuleRecord(ctx, *name, opts...)
	case "policy":
		result, err = service.DeleteFirewallPolicyRecord(ctx, *name, opts...)
	case "group":
		result, err = service.DeleteFirewallGroupRecord(ctx, *name, opts...)
	}
	if err != nil {
		log.Fatal(err)
	}
	value, err := json.Marshal(result.Deleted)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(value))
}
```

## References

- [Policy Sample](https://docs.openstack.org/neutron-fwaas/2025.2/configuration/policy-sample.html)
