# Cloud Project Group Get Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--kind", choices=["project", "group"], default="project")
    parser.add_argument("--name", required=True)
    parser.add_argument("--domain-id", default="")
    parser.add_argument("--filters", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    domain = args.domain_id or None
    if args.kind == "project":
        value = conn.get_project(args.name, filters=filters, domain_id=domain)
    else:
        value = conn.get_group(args.name, filters=filters, domain_id=domain)
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))

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
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func main() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	kind := flag.String("kind", "project", "project or group")
	name := flag.String("name", "", "literal name or ID")
	domain := flag.String("domain-id", "", "optional domain UUID")
	filters := flag.String("filters", "", "optional JSON filters")
	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	var value json.RawMessage
	switch *kind {
	case "project":
		var options []identity.ProjectGetOption
		if *domain != "" {
			options = append(options, identity.WithProjectGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithProjectGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetProjectRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	case "group":
		var options []identity.GroupGetOption
		if *domain != "" {
			options = append(options, identity.WithGroupGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithGroupGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetGroupRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	default:
		log.Fatal("-kind must be project or group")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`get_project/get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [Project](v3/projects/python-record-reads.md)
- [Group](v3/groups/python-records.md)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py)
- [Group](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
