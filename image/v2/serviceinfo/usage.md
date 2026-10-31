# Usage

`GetUsageInfo(ctx)` reads usage and limits for the authenticated project as a singleton. It preserves dynamic resource entries, exact int64 values, omitted/null values, original response fields, headers, and status. Disabled server reporting can produce empty usage information. There is no resource ID, list, mutation, name lookup, or status wait.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [records_prepare.go](records_prepare.go)
- [options.go](options.go)
- [usage_model.go](usage_model.go)
- [store_records.go](store_records.go)
- [records.go](records.go)
- [python_store_details.go](python_store_details.go)
- [import_record.go](import_record.go)
- [store_import_identity.go](store_import_identity.go)
- [api.go](api.go)
- [model.go](model.go)
- [prepare.go](prepare.go)
- [usage.go](usage.go)
- [records_options.go](records_options.go)
- [usage_options.go](usage_options.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverUsage(ctx context.Context, conn *openstack.Connection) (*serviceinfo.UsageInfo, error) {
    service, err := conn.ImageV2(ctx)
    if err != nil {
        return nil, err
    }
    return service.ServiceInfo.GetUsageInfo(ctx)
}
```

```python
import json
from urllib.request import Request, urlopen

def get_usage_http(image_endpoint, token):
    request = Request(
        image_endpoint.rstrip("/") + "/info/usage",
        headers={"X-Auth-Token": token, "Accept": "application/json"},
        method="GET",
    )
    with urlopen(request) as response:
        if response.status != 200:
            raise RuntimeError(f"unexpected usage status: {response.status}")
        return json.loads(response.read())
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#quota-usage)
- [Yoga 24.0.0 release notes](https://docs.openstack.org/releasenotes/glance/yoga.html)
- [router](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L607-L610)
- [Keystone](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/quota/keystone.py#L147-L170)
- [usage schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L235-L236)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L479-L480)
- [usage controller·serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L197-L213)
- [Usage Core Test](usage_core_test.go)
- [Usage Contracts Test](usage_contracts_test.go)
- [Connection Image Usage Test](../../../connectiontest/connection_image_usage_test.go)
- [Glance Serviceinfo Test](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)
