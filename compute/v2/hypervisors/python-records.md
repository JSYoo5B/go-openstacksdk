# Hypervisor listing and lookup

`ListRecords` follows `compute.hypervisors`: summary by default, `WithRecordDetails(true)` for detail, and the pre-2.53 hostname search route. `FindRecord` follows `find_hypervisor`: GET first, then detail listing after a clean 400/403/404. `WithRecordDetails(false)` changes only its fallback to summary. Every matching row counts, including duplicate IDs; missing defaults to nil and `WithRecordIgnoreMissing(false)` makes it an error.

The library owns alias projection, dictionary/list descriptors, passive response IDs and literal ID comparison, pagination and observed response evidence. Nova discovery chooses a maximum of 2.88 when no version is selected. A hostname pattern separately checks advertised bounds and the selected version before choosing the legacy search route. Explicit selected versions remain explicit.

```python
import openstack
conn = openstack.connect()
rows = list(conn.compute.hypervisors(details=False, with_servers=True))
node = conn.compute.find_hypervisor("node", ignore_missing=False)
print(len(rows), node.name)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
)
func main() {
    provider := &gophercloud.ProviderClient{}
    provider.UseTokenLock()
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "compute", Endpoint: os.Getenv("OS_COMPUTE_ENDPOINT")}
    api := hypervisors.New(client)
    rows, err := api.AllRecords(context.Background(), hypervisors.WithRecordQuery("with_servers", true))
    if err != nil { panic(err) }
    node, err := api.FindRecord(context.Background(), "node", hypervisors.WithRecordIgnoreMissing(false))
    if err != nil { panic(err) }
    fmt.Println(len(rows), string(node.Resource.Body["name"]))
}
```

Go uses scalar string identities, JSON options and immutable response views rather than Python Resource instances. Unknown list kwargs are discarded by the declared profile; arbitrary Python objects are outside JSON options. `ProxyListRecords` preserves Source JMESPath evaluation against an opaque, unconsumed generator using the shared interpreter. `[*].name` returns JSON null, `length(@)` fails, and `@` preserves a `*RecordGenerator` whose `Records` performs HTTP only when iterated. `ListRecords` and `AllRecords` have fixed row result types; nonempty textual expressions direct callers to `ProxyListRecords`. This is an API mapping, not full SDK parity. Discovery/HTTP fixtures and example compilation do not establish live-cloud behavior.

A failure while preparing version discovery returns an error before a resource record exists. `Discovery` evidence is attached to successful resource records; this binding does not return a discovery-only receipt when preparation fails. The error retains the underlying failure, and this boundary is not live-cloud validation.
