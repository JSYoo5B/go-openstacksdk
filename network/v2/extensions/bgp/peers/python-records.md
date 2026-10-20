# BGP peers: listing and lookup

`ListRecords` and `AllRecords` follow `network.bgp_peers`. This class derives directly from Resource: only `limit` and `marker` are server query parameters. Declared Body filters such as `name` and `project_id` run locally; unknown list kwargs are ignored before encoding. No automatic name hint is sent.

`FindRecord` follows `find_bgp_peer`: one literal member GET, then list fallback only for a clean 400/403/404. Complete list traversal detects duplicates, including repeated IDs. Missing returns nil by default; `WithRecordFindIgnoreMissing(false)` returns an error. Numeric response IDs remain passive JSON and do not equal a string lookup identity.

`project_id` falls back to `tenant_id` when absent. Unknown Body extensions survive in the owned resource view. No tag/revision/header defaults are inferred from NetworkResource. Response fields retain Source's untyped values. `Wire` and `Envelope` retain actual response data separately from projected aliases and constructor seed values.

```python
import openstack
conn = openstack.connect()
rows = list(conn.network.bgp_peers(name="routing", project_id="project"))
row = conn.network.find_bgp_peer("routing", ignore_missing=False)
print(len(rows), row.id)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
)
func main() {
    provider := &gophercloud.ProviderClient{}
    provider.UseTokenLock()
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "network", Endpoint: os.Getenv("OS_NETWORK_ENDPOINT")}
    api := peers.New(client)
    rows, err := api.AllRecords(context.Background(), peers.WithRecordListQuery("name", "routing"), peers.WithRecordListQuery("project_id", "project"))
    if err != nil { panic(err) }
    row, err := api.FindRecord(context.Background(), "routing", peers.WithRecordFindIgnoreMissing(false))
    if err != nil { panic(err) }
    fmt.Println(len(rows), string(row.Resource.Body["id"]))
}
```

`ProxyListRecords` reuses the shared Source-compatible JMESPath interpreter and opaque generator identity: `[*].name` returns null without HTTP; `@` preserves `*RecordListGenerator`; `length(@)` fails before iteration. Consume `Records` to perform physical requests. The fixed-row list API directs nonempty textual expressions to `ProxyListRecords`.

Find also supports raw fetch controls (`requires_id`, `base_path`, `skip_cache`, `error_message`, `resource_response_key`, `headers`) through owned JSON options. Raw `headers` is a GET query value and becomes HTTP headers only on fallback; typed headers explicitly apply to both stages. Header validation, cancellation and physical read/Close failures remain terminal. Use `WithRecordFindOpts` / `WithRecordListOpts` for reusable snapshots.

Go uses explicit scalar identities and finite JSON inputs rather than mutable Python Resource objects or arbitrary Python callables. Routes stay within the selected service; unsafe route and auth overrides are rejected. Shared session/Resource contracts, arbitrary Requests codecs, and live-cloud/release validation remain separate unfinished work. These APIs are mappings, not a full parity claim.

Existing native `List`/`All` entry points retain their earlier one-request/no-wire-query behavior and native paging limitations. These retained native observations do not describe the new `ListRecords`, `AllRecords` or `FindRecord` Source profiles.
