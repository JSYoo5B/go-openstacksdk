# Trunk Address Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

with openstack.connect(cloud="devstack") as conn:
    trunk = conn.network.find_trunk(
        "application-trunk", ignore_missing=False,
        project_id="project-id", is_admin_state_up=False,
        fields=["id", "name", "admin_state_up"],
    )
    address_group = conn.network.find_address_group(
        "application-addresses", ignore_missing=False,
        project_id="project-id", addresses=["10.0.0.0/24"],
    )
    print(json.dumps(trunk.to_dict(), indent=2))
    print(json.dumps(address_group.to_dict(), indent=2))
```

See `Example_trunkAddressFind` in [example_trunk_address_find_test.go](example_trunk_address_find_test.go), compiled by `go test`.
