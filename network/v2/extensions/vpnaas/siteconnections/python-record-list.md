# VPN IPsec site connections: Python and Go records

`siteconnections.API.ListRecords` maps pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` `network.vpn_ipsec_site_connections(**query)`. The SDK classifies query parameters, applies local Body filters to each received row, follows all pages and owns descriptor conversion. Existing generated `List` retains its native typed contract.

| Python | Go |
|---|---|
| `vpn_ipsec_site_connections()` | `ListRecords(ctx)` lazy iterator |
| `list(vpn_ipsec_site_connections())` | `AllRecords(ctx)` |
| `status="ACTIVE", mtu=1500` | `WithRecordListQuery` sends declared server queries |
| `interval=30, peer_cidrs=["10.0.0.0/24"]` | same option applies local descriptor filters |
| `paginated=False`, `max_items=10` | `WithRecordListPaginated(false)`, `WithRecordListMaxItems(10)` |
| deprecated `jmespath_filters` | `ProxyListRecords` and `WithRecordListJMESPath` |

Declared server queries are auth_mode, description, fields, ikepolicy_id, ipsecpolicy_id, initiator, local_ep_group_id, peer_address, local_id, mtu, name, peer_id, project_id, psk, peer_ep_group_id, route_mode, vpnservice_id, status, admin_state_up/is_admin_state_up, limit and marker. Declared Body attributes outside that table, including action, dpd, interval, peer_cidrs, timeout and id, filter the loaded descriptor view locally. Unknown keys are discarded before their values are JSON-marshaled. `tenant_id` is a fallback response alias for project_id, not a declared query or standalone local Body attribute; a tenant_id keyword is discarded.

Missing fields remain null. The documented server defaults hold/30/120/static are not synthesized into a missing response. `dpd` uses dictionary conversion, `peer_cidrs` wraps scalar input into a list, interval/mtu/timeout use Python integer descriptor semantics, and is_admin_state_up uses Python truthiness: nonempty string `"false"` is true. Raw response fields and JSON numeric literals remain in Wire independently from the descriptor view. A declared query such as mtu is sent to the server rather than reapplied as a local descriptor filter.

The following standalone programs connect using clouds.yaml and print connections with server status ACTIVE and local interval 30.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        for row in conn.network.vpn_ipsec_site_connections(status="ACTIVE", interval=30):
            print(json.dumps(row.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
)

func run(ctx context.Context, cloud string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    for row, err := range service.VPNSiteConnections.ListRecords(ctx,
        siteconnections.WithRecordListQuery("status", "ACTIVE"),
        siteconnections.WithRecordListQuery("interval", 30)) {
        if err != nil { return err }
        if err := json.NewEncoder(os.Stdout).Encode(row.Resource.Body); err != nil { return err }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    timeout := flag.Duration("timeout", time.Minute, "whole list deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Existing selected clients use `siteconnections.New(client)`. `RecordListOpts` owns raw JSON/map/pointer values and each option application gets a fresh snapshot. Explicit headers, microversion and literal service-relative base path apply only to the owned client copy. Constructor controls such as cls/resource_type/session/self are preflight errors; dedicated factories handle header/version/base path controls. Conflict restoration uses `WithRecordListConflictingAttrs` with ordered JSON.

Pagination defaults true, max_items defaults zero/unbounded, and no limit is invented. Source plural links, top-level next, HTTP Link and explicit-limit marker fallback use the shared Neutron pager. Marker and max_items accounting use physical rows before local filtering; a filtered page cannot hide the last physical ID or prematurely terminate paging. Explicit-limit marker fallback continues after a nonempty short page until the next empty page. Stopping iterator consumption avoids later requests. `AllRecords` discards partially collected records on error, while iterator consumers retain records already yielded.

`SiteConnectionRecord` separates Resource, Wire, raw Envelope, Header and StatusCode. Unknown response fields remain observable without requiring a caller-defined decoder. Failed HTTP status/body/headers and context causes remain in the error chain. Shared source guards reject route changes and unsafe continuation targets. The leaf owns Source status acceptance of final 100..399; net/http intermediate/redirect behavior and live Neutron authorization are separate validation.

Proxy JMESPath operates on an opaque Python generator, not a pre-collected JSON list. `@` returns `*RecordListGenerator` without HTTP; consume its Records explicitly. Projection `[].id` and `type(@)` yield JSON null without HTTP, and `length(@)` fails without HTTP. `RecordProxyListResult.Value` can contain that opaque generator, while JSON is populated only for serializable results. Ordinary `ListRecords`/`AllRecords` reject nonempty expressions because their result type is resource records. This mirrors the pinned upstream generator behavior rather than adding an eager projection API.

Go provides a finite JSON profile and typed options instead of mutable Python classes, arbitrary objects or dynamic binding. Existing shared name/header/continuation validation and request safety limits remain in force. These operation mappings do not claim full SDK Resource/session lifecycle parity or live VPN service validation. Focused tests reuse nativeVPNAPI/nativeVPNWire and the Gophercloud public testhelper fixture for descriptor projection, server/local filters, full paging, opaque expressions, owned inputs, late HTTP errors, cancellation and source guards. Root runs Go focused/full checks and the example build; Python example syntax is checked separately.
