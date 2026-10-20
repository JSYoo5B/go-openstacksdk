# Source-shaped segments lists

```python
import openstack

conn = openstack.connect()
for record in conn.network.segments(network_type="vlan"):
    print(record.id)
opaque = conn.network.segments(jmespath_filters="@")
print(opaque)
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    provider := &gophercloud.ProviderClient{}
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{
        ProviderClient: provider, Type: "network",
        Endpoint: os.Getenv("OS_NETWORK_ENDPOINT"),
    }
    api := segments.New(client)
    for record, err := range api.ListRecords(context.Background(),
        segments.WithRecordListQuery("network_type", "vlan")) {
        if err != nil { panic(err) }
        fmt.Println(string(record.Resource.Body["id"]))
    }
    result, err := api.ProxyListRecords(context.Background(),
        segments.WithRecordListJMESPath("@"))
    if err != nil { panic(err) }
    fmt.Printf("%T\n", result.Value)
}
```

`ListRecords` is lazy and starts HTTP when consumed. `AllRecords` returns an
owned slice, discarding partial collected results on error. Native `List` keeps
its existing typed transport model. Declared server QueryParameters are sent
with Requests-style JSON query spelling. Other declared Body fields filter
received rows locally: inherited id (all Segment-specific Body fields are declared server queries). Unknown query
kwargs are discarded before invoking a JSON marshaler. Unknown response Body
extensions remain in Resource and Wire because these Source classes opt in to
unknown Body attributes. HTTP envelope, headers and status remain independent.

`WithRecordListLimit`, `Marker`, `MaxItems` and `Paginated` own paging controls;
resource links, top-level next and HTTP Link continuations use the shared pager.
Raw row caps apply before local Body filtering. Breaking an iterator avoids
requesting later pages. Base paths stay inside the selected service, and options
copy maps and raw JSON. Source snapshots and cancellation remain guarded.

`ProxyListRecords` preserves the actual deprecated Proxy JMESPath behavior:
Resource.list is an opaque generator, not a pre-collected array. `@` returns the
generator without HTTP, `[0]` returns JSON null without HTTP, and `length(@)`
fails before HTTP. JSON is populated only for serializable results; nested
opaque generators stay in Value. Use the dedicated Proxy method for expressions;
ListRecords/AllRecords reject a truthy expression to keep their result types.
`WithRecordListConflictingAttrs` restores Proxy attrs once using the shared
normalizer; restored expressions cannot replace the original Proxy expression.

The API uses concrete options and a finite JSON domain. Dynamic Python classes,
sessions and mutable Resource instances are replaced by SDK-owned profiles;
shared SDK-R1/C1/S1, codec/runtime and real-cloud/release checks remain open.
Local tests and compiled examples do not prove live-cloud interoperability.

segmentation_id is the only typed Segment field (Source integer descriptor). Segment inherits Resource directly, so no NetworkResource tag, revision or Header descriptors are introduced.
