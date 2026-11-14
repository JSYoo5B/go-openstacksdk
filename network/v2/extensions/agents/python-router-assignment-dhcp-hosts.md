# Python Router Assignment Dhcp Hosts

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=("list", "update"), default="list")
parser.add_argument("--network", default="")
parser.add_argument("--agent", default="")
parser.add_argument("--router", default="")
args = parser.parse_args()
with openstack.connect() as conn:
    if args.mode == "list":
        for row in conn.network.network_hosting_dhcp_agents(args.network, is_alive=True, max_items=20):
            print(json.dumps(row.to_dict(), indent=2))
    else:
        value = conn.network.update_router_in_agent(args.agent, args.router, ha_chassis_priority=0)
        print(json.dumps(value, indent=2))
```

See `Example_routerAssignmentDhcpHosts` in [example_router_assignment_dhcp_hosts_test.go](example_router_assignment_dhcp_hosts_test.go), compiled by `go test`.
