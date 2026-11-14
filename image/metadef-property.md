# Metadef Property

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property("hw_cpu_policy", "OS::Compute::Libvirt")
print(value.to_dict())
```

```go
service, err := conn.Image(ctx)
if err != nil { return err }
scope, err := service.API.MetadefProperties.InNamespace(ctx, "OS::Compute::Libvirt")
if err != nil { return err }
value, err := scope.Get(ctx, "hw_cpu_policy")
if err != nil { return err }
```

```python
# TypeError before HTTP: Proxy._get already binds resource_type.
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt", resource_type="OS::Nova::Server"
)
```

```go
value, err := scope.Get(ctx, "hw_cpu_policy",
    request.Custom(func(c *metadefproperties.GetOpts) error { v := "OS::Nova::Server"; c.ResourceType = &v; return nil }))
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
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "hw_cpu_policy", "literal property name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name); err != nil { log.Fatal(err) }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{"status_code": accepted.StatusCode,
            "header": accepted.Header, "body": string(accepted.Body)}
    case errors.As(err, &unexpected):
        receipt = map[string]any{"status_code": unexpected.Actual,
            "header": unexpected.ResponseHeader, "body": string(unexpected.Body)}
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    value, err := scope.Get(ctx, name)
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(value.Body, "", "  ")
    if err != nil { return err }
    fmt.Printf("namespace=%q requested_name=%q status_code=%d\n%s\n",
        scope.NamespaceName(), name, value.StatusCode, data)
    return nil
}
```

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [core](v2/metadefproperties/core_test.go)
- [Contracts Test](v2/metadefproperties/contracts_test.go)
- [Options Test](v2/metadefproperties/options_test.go)
- [Connection](../connectiontest/connection_image_metadef_properties_test.go)
- [Get Mapping Test](v2/metadefproperties/get_mapping_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931)
- [MetadefProperty](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79)
- [Proxy._get_resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L605)
- [Proxy._get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1849)
- [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
