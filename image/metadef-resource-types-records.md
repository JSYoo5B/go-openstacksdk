# Metadef Resource Types Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_resource_types(
    limit=10, max_items=20, paginated=False
):
    print(value.to_dict())

for value in conn.image.metadef_resource_type_associations(
    namespace, limit=10, max_items=20, name="OS::Nova::Server", paginated=False
):
    print(value.to_dict())
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "", "optional association name filter")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", false, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name, *limit, *maxItems, *paginated); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *types.Record) error {
    if record == nil { return nil }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource,
        "wire": record.Wire,
        "envelope": record.Envelope,
        "header": record.Header,
        "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    api := service.API.MetadefResourceTypes
    options := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        request.Header("X-Request-Source", "metadef-records-example"),
        request.Paginated(paginated),
    }
    for record, readErr := range api.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    scope, err := api.InNamespace(ctx, namespace)
    if err != nil { return err }
    scopedOptions := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{
            Limit: limit, MaxItems: maxItems,
            Filters: associationFilters(name),
        }),
        request.Header("X-Request-Source", "metadef-records-example"),
        request.Paginated(paginated),
    }
    records, readErr := scope.AllRecords(ctx, scopedOptions...)
    for _, record := range records {
        if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}

func associationFilters(name string) []resource.ListOption {
    if name == "" { return nil }
    return []resource.ListOption{resource.WithFilter("name", name)}
}
```

## References

- [Metadefresourcetypes](v2/metadefresourcetypes/README.md)
- [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L805-L844)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)
- [Resource class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2459)
- [List Records Test](v2/metadefresourcetypes/list_records_test.go)
- [Connection Image Metadef Records Test](../connectiontest/connection_image_metadef_records_test.go)
- [leaf core fixture](v2/metadefresourcetypes/core_test.go)
- [Contracts Test](v2/metadefresourcetypes/contracts_test.go)
- [List Control Test](../internal/rest/list_control_test.go)
- [List Short Page Test](../internal/rest/list_short_page_test.go)
- [Filter Test](../internal/jsonfilter/filter_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
