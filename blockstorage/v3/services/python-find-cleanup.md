# Service lookup and worker cleanup

The owned finder reads `os-services` directly, without a member GET. It matches
stringified IDs or the declared name (falling back to binary only when name is
absent), applies host matching, and finishes the inventory before returning so
late failures and duplicate matches remain errors. Missing defaults to nil;
`WithFindRecordIgnoreMissing(false)` requests an error. A selected Cinder version
is preserved; an unset version negotiates up to the Source Service maximum 3.32.

```python
import openstack

conn = openstack.connect()
service = conn.block_storage.find_service("cinder-volume", host="host@lvm")
cleanup = conn.block_storage.cleanup_service_workers(
    binary=None, host=None, cluster_name=None, is_up=False,
)
print(service, cleanup)
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    blockstorage "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    provider := &gophercloud.ProviderClient{}
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{
        ProviderClient: provider, Type: "volumev3",
        Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"), Microversion: "3.24",
    }
    sdk := blockstorage.New(client)
    found, err := sdk.Services.FindRecord(context.Background(), "cinder-volume",
        services.WithFindRecordQuery("host", "host@lvm"))
    if err != nil { panic(err) }
    fmt.Println(found)
    result, err := sdk.Services.CleanupServiceWorkers(context.Background(),
        services.WithCleanupWorkerFilter("is_up", false))
    if err != nil { panic(err) }
    fmt.Println(string(result.JSON))
}
```

`CleanupServiceWorkers` sends all eight keys. Go binds omitted binary, host and
cluster_name to JSON null; the pinned Python signature requires those three
keywords even though its prose describes them as optional. `is_up` defaults to
false; the other filters default to null. Explicit null/false/zero JSON values
remain distinct. Unknown keys fail before HTTP. The version must be at least
3.24; an unset version negotiates using the Service ceiling 3.32. The result
preserves arbitrary JSON and actual response/discovery evidence.

Finder options reuse the shared Source-shaped list engine for host/binary
server queries, declared Body filters, limit/marker, max_items/paginated, raw
headers and literal service-relative base_path/list_base_path. The Service.find
microversion formal is consumed but intentionally ignored, as in the pinned
Source override; version selection remains the Service class policy. Truthy
list_base_path overrides base_path. __conflicting_attrs and jmespath_filters are
ordinary unknown Resource.list kwargs here, so they do not restore Proxy
arguments or run expressions. Bound session/class replacements remain rejected
under the shared Go JSON profile. Go options and immutable record evidence
replace mutable Resource instances. Existing native List and service actions
retain their own behavior. No live-cloud validation is claimed by these examples.
