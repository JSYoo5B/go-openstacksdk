# Updating a QoS policy with Source attributes

`UpdateRecord` implements the fresh scalar-ID form of `network.update_qos_policy`. It owns JSON attributes and sends one wrapped `PUT qos/policies/{id}`. Only supplied attributes become dirty. Empty attributes and `if_revision` alone return a seeded record without HTTP or invented response evidence.

QoSPolicy derives from Resource and TagMixin, without NetworkResource. Canonical `is_shared` and wire `shared` address the same field; later supplied aliases win. Values remain raw in the request: strings, explicit null, false, scalar rules/tags and unknown Body extensions are not silently replaced by projected defaults. Descriptors affect only `PolicyRecord.Resource`: boolean truthiness, list casts, tags' omitted `[]`, project/tenant alias and explicit-null presence. `Wire`, `Envelope`, `Header` and `StatusCode` preserve the actual response separately. Partial responses overlay the raw request seed; a canonical-looking unknown response member cannot override a wire-backed getter.

```python
import openstack
conn = openstack.connect()
policy = conn.network.update_qos_policy("policy-id", name="gold", is_shared=True, description=None)
print(policy.id, policy.is_shared)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
)
func main() {
    provider := &gophercloud.ProviderClient{}
    provider.UseTokenLock()
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "network", Endpoint: os.Getenv("OS_NETWORK_ENDPOINT")}
    row, err := policies.New(client).UpdateRecord(context.Background(), "policy-id",
        policies.WithRecordUpdateAttribute("name", "gold"),
        policies.WithRecordUpdateAttribute("is_shared", true),
        policies.WithRecordUpdateAttribute("description", nil))
    if err != nil { panic(err) }
    fmt.Println(string(row.Resource.Body["id"]), string(row.Resource.Body["is_shared"]))
}
```

`WithRecordUpdateOpts` snapshots raw attribute bytes, typed headers and a microversion. Bulk maps use sorted keys; individual attribute options retain insertion order. A scalar-ID `id` attribute and bound Proxy/Resource constructor arguments fail before HTTP. `if_revision` is consumed without generating If-Match because this class is not NetworkResource. Ordinary `if_match` remains a Body extension. The separate native update API keeps its explicit revision-header capability.

Returned records bind the existing trusted `UpdateCloudRecord` lane, which compares its restricted cloud attributes against a private borrowed snapshot and preserves no-op receipt/equality behavior. Exported field edits cannot corrupt that snapshot. This is an explicit Go lifecycle API, not arbitrary mutable Python Resource identity or cache semantics.

The new scalar profile accepts only nonblank safe string identities and finite UTF-8 JSON values. Raw `base_path` is explicitly unsupported by this fixed-route mutation helper. Typed transport controls are Go-owned conveniences. Arbitrary Python objects, Requests codec/sniffing behavior, shared session and Resource contracts, live-cloud checks and release validation remain separate gaps. Local HTTP tests do not establish live-cloud parity. Native `Update` and `CreateWithAttributes` retain their earlier behavior.
