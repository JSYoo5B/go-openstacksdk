# Schema Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace_schema = conn.image.get_metadef_namespace_schema()
task_schema = conn.image.get_task_schema()
print(namespace_schema.to_dict())
print(task_schema.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloudName); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *image.SchemaRecord) error {
    if record == nil { return nil }
    data, err := json.MarshalIndent(map[string]any{
        "kind": record.Kind, "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func printFailure(err error) error {
    var data map[string]any
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    switch {
    case errors.As(err, &accepted):
        data = map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body": string(accepted.Body),
        }
    case errors.As(err, &unexpected):
        data = map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body": string(unexpected.Body),
        }
    default:
        return nil
    }
    raw, printErr := json.MarshalIndent(data, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(raw))
    return nil
}

func run(ctx context.Context, cloudName string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    options := []image.GetSchemaOption{
        request.Header("X-Request-Source", "schema-record-example"),
    }
    for _, kind := range []image.SchemaKind{image.SchemaMetadefNamespace, image.SchemaTask} {
        record, readErr := service.GetSchemaRecord(ctx, kind, options...)
        if printErr := printRecord(record); printErr != nil {
            return errors.Join(readErr, printErr)
        }
        if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
    }
    return nil
}
```

## References

- [Schemas](schemas.md)
- [Schema Records Test](schema_records_test.go)
- [Connection Image Schema Records Test](../connectiontest/connection_image_schema_records_test.go)
- [Schemas Core Test](schemas_core_test.go)
- [Schemas Contracts Test](schemas_contracts_test.go)
- [Schemas Options Test](schemas_options_test.go)
- [Connection](../connectiontest/connection_image_schemas_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1951-L2160)
- [Schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/schema.py#L17-L26)
- [MetadefSchema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_schema.py#L17-L30)
