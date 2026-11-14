# Metadef Object Namespace Record Lists

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_namespaces(
    visibility="private", owner="project-id", is_protected=False,
    limit=10, max_items=20, paginated=True
):
    print(value.to_dict())

for value in conn.image.metadef_objects(namespace):
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
    namespaces "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
    objects "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal object parent")
    namespaceFilter := flag.String("namespace-filter", "", "optional namespace Body filter")
    owner := flag.String("owner", "", "optional namespace owner Body filter")
    objectName := flag.String("object-name", "", "optional object name Body filter")
    visibility := flag.String("visibility", "", "optional namespace server query")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", true, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    err := run(ctx, *cloud, *namespace, *namespaceFilter, *owner, *objectName,
        *visibility, *limit, *maxItems, *paginated)
    if err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            if printErr := printJSON(map[string]any{
                "error_status": response.StatusCode, "error_header": response.Header,
                "error_body": response.Body,
            }); printErr != nil { log.Print(printErr) }
        }
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, namespaceFilter, owner,
    objectName, visibility string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    namespaceOptions := []namespaces.RecordListOption{
        namespaces.WithRecordListOpts(namespaces.RecordListOpts{MaxItems: maxItems}),
        request.Header("X-Request-Source", "metadef-record-lists-example"),
        request.Paginated(paginated),
    }
    if limit > 0 {
        namespaceOptions = append(namespaceOptions, request.Limit(limit))
    }
    if visibility != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("visibility", visibility))
    }
    if namespaceFilter != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("namespace", namespaceFilter))
    }
    if owner != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("owner", owner))
    }
    for record, readErr := range service.MetadefNamespaces.ListRecords(ctx, namespaceOptions...) {
        if readErr != nil { return readErr }
        if err := printJSON(record); err != nil { return err }
    }
    scope, err := service.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return err }
    objectOptions := []objects.RecordListOption{
        objects.WithRecordListOpts(objects.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        request.Header("X-Request-Source", "metadef-record-lists-example"),
        request.Paginated(paginated),
    }
    if objectName != "" {
        objectOptions = append(objectOptions, objects.WithRecordListFilter("name", objectName))
    }
    records, readErr := scope.AllRecords(ctx, objectOptions...)
    for _, record := range records {
        if err := printJSON(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}
```

## References

- [Metadefobjects](v2/metadefobjects/README.md)
- [namespace](v2/metadefnamespaces/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1449-L1613)
- [Metadef Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_object.py)
- [namespace class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2436)
- [jmespath.py1.0.1 interpreter](https://github.com/jmespath/jmespath.py/blob/1.0.1/jmespath/visitor.py)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
