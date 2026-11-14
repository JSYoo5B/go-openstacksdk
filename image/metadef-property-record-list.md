# Metadef Property Record List

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
for value in conn.image.metadef_properties(
    "OS::Compute::Libvirt", min_length=0
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
    properties "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    maxRows := flag.Int("max-rows", 0, "maximum consumed dictionary entries")
    minLength := flag.Int("min-length", 0, "local canonical min_length filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *maxRows, *minLength); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *properties.Record) error {
    data, err := json.MarshalIndent(map[string]any{
        "key": record.Key, "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
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

func run(ctx context.Context, cloud, namespace string, maxRows, minLength int) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    records, readErr := scope.AllRecords(ctx,
        properties.WithRecordListOpts(properties.RecordListOpts{MaxItems: maxRows}),
        properties.WithRecordListFilter("min_length", minLength),
        request.Header("X-Request-Source", "metadef-property-list-example"),
    )
    for _, record := range records {
        if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
    }
    if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
    return nil
}
```

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1876-L1898)
- [Property list override](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)
- [List Records Test](v2/metadefproperties/list_records_test.go)
- [Connection Image Metadef Property List Records Test](../connectiontest/connection_image_metadef_property_list_records_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
