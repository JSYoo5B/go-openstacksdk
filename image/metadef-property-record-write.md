# Metadef Property Record Write

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
id = "hw_cpu_policy"

created = conn.image.create_metadef_property(
    namespace, name=id, type="string", title="CPU Policy",
    description="CPU allocation policy", is_readonly=False, min_length=0,
)
updated = conn.image.update_metadef_property(
    id, namespace, name=id, type="string", title="CPU Policy",
    description="Updated CPU allocation policy", is_readonly=False, min_length=0,
)
local = conn.image.update_metadef_property(id, namespace)
print(created.to_dict(), updated.to_dict(), local.to_dict())
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
    properties "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    id := flag.String("id", "hw_cpu_policy", "property name")
    action := flag.String("action", "snapshot", "snapshot, create, or update")
    title := flag.String("title", "CPU Policy", "property title")
    description := flag.String("description", "CPU allocation policy", "property description")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *id, *action, *title, *description); err != nil {
        log.Fatal(err)
    }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body": string(accepted.Body),
        }
    case errors.As(err, &unexpected):
        receipt = map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body": string(unexpected.Body),
        }
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, id, action, title, description string) error {
    if action != "snapshot" && action != "create" && action != "update" {
        return fmt.Errorf("unknown action %q", action)
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    attributes := map[string]any{
        "name": id, "type": "string", "title": title,
        "description": description, "is_readonly": false, "min_length": 0,
    }
    var record *properties.Record
    switch action {
    case "create":
        record, err = scope.CreateRecord(ctx,
            properties.WithRecordCreateAttributes(attributes),
            request.Header("X-Request-Source", "metadef-property-write-example"),
        )
    case "update":
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            properties.WithRecordUpdateAttributes(attributes),
            request.Header("X-Request-Source", "metadef-property-write-example"),
        )
    default:
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            request.Header("X-Request-Source", "metadef-property-write-example"),
        )
    }
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [Metadef Property Record List](metadef-property-record-list.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1841)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L811)
- [constructor·attribute collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336)
- [commit no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2017)
- [descriptor get/set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208-L291)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
