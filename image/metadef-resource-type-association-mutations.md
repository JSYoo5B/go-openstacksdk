# Metadef Resource Type Association Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
resource_type = "OS::Nova::Server"

created = conn.image.create_metadef_resource_type_association(
    namespace, name=resource_type, prefix="hw:", properties_target="metadata",
)
print(created.to_dict())
conn.image.delete_metadef_resource_type_association(
    resource_type, namespace, ignore_missing=False,
)
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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    resourceType := flag.String("resource-type", "OS::Nova::Server", "resource type name")
    action := flag.String("action", "list", "list, create, or delete")
    prefix := flag.String("prefix", "hw:", "association property prefix")
    target := flag.String("properties-target", "metadata", "association target")
    strictMissing := flag.Bool("strict-missing", false, "return an error for deletion404")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *resourceType, *action, *prefix, *target, *strictMissing); err != nil {
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func printRecord(record *types.Record) error {
    return printJSON(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    })
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    switch {
    case errors.As(err, &accepted):
        return printJSON(map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body_base64": accepted.Body,
        })
    case errors.As(err, &unexpected):
        return printJSON(map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body_base64": unexpected.Body,
        })
    default:
        return nil
    }
}

func run(ctx context.Context, cloud, namespace, resourceType, action, prefix, target string, strictMissing bool) error {
    if action != "list" && action != "create" && action != "delete" {
        return fmt.Errorf("unknown action %q", action)
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefResourceTypes.InNamespace(ctx, namespace)
    if err != nil { return err }
    switch action {
    case "create":
        record, createErr := scope.CreateRecord(ctx,
            types.WithRecordCreateAttributes(map[string]any{
                "name": resourceType, "prefix": prefix, "properties_target": target,
            }),
            request.Header("X-Request-Source", "association-example"),
        )
        if createErr != nil { return errors.Join(createErr, printFailure(createErr)) }
        return printRecord(record)
    case "delete":
        ack, deleteErr := scope.DeleteRecord(ctx, types.RecordRequest{ID: resourceType},
            request.IgnoreMissing(!strictMissing),
            request.Header("X-Request-Source", "association-example"),
        )
        if ack != nil {
            if err := printJSON(ack); err != nil { return errors.Join(deleteErr, err) }
        }
        if deleteErr != nil { return errors.Join(deleteErr, printFailure(deleteErr)) }
        return nil
    default:
        records, readErr := scope.AllRecords(ctx,
            request.Paginated(false),
            request.Header("X-Request-Source", "association-example"),
        )
        for _, record := range records {
            if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
        }
        if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
        return nil
    }
}
```

## References

- [Metadef Resource Types Records](metadef-resource-types-records.md)
- [Metadefresourcetypes](v2/metadefresourcetypes/README.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1711-L1761)
- [association class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py#L34-L61)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L609)
- [dirty request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336)
- [Response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
- [Delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2154)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
