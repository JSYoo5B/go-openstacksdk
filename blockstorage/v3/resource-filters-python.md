# Resource Filters Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import jmespath
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.33")
    parser.add_argument("--resource", default=None)
    parser.add_argument("--single-page", action="store_true")
    parser.add_argument("--expression", default="")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    rows = list(conn.block_storage.resource_filters(
        resource=args.resource,
        paginated=not args.single_page,
    ))
    data = [row.to_dict() for row in rows]
    # Explicit repair: the pinned Proxy passes a generator to JMESPath.
    value = jmespath.search(args.expression, data) if args.expression else data
    print(json.dumps(value, indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.33", "selected Cinder version")
    resourceName := flag.String("resource", "", "resource name; empty omits query")
    single := flag.Bool("single-page", false, "stop after the first page")
    expression := flag.String("expression", "", "optional JMESPath result")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    filters := resourcefilters.New(service.RawClient())
    options := []resourcefilters.RecordListOption{
        request.Custom(func(c *resourcefilters.RecordListOpts) error { c.Control = resource.ListControl{SinglePage: *single}; return nil }),
        resourcefilters.WithRecordListJMESPath(*expression),
    }
    if *resourceName != "" {
        options = append(options, resourcefilters.WithRecordListResource(*resourceName))
    }
    result, err := filters.ListResult(ctx, options...)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(result.Value))
}
```

## References

- [resource_filters Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3620-L3628)
- [ResourceFilter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/resource_filter.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#list-resource-filters)
- [ResourceFiltersController.index](https://github.com/openstack/cinder/blob/master/cinder/api/v3/resource_filters.py)
- [Python Contracts Test](resourcefilters/python_contracts_test.go)
- [Python Proxy Contracts Test](resourcefilters/python_proxy_contracts_test.go)
