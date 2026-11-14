# Finding Identities

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    storagev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindApplicationResources(ctx context.Context, compute *computev2.Service,
    storage *storagev3.Service, network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", strict)
    if err != nil { return err }
    port, err := network.Ports.FindIdentity(ctx, "web-port", strict)
    if err != nil { return err }
    _, _, _ = server, volume, port
    return nil
}
```

```go
package example

import (
    "context"

    dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
    lbv2 "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindScopedResources(ctx context.Context, dns *dnsv2.Service,
    loadbalancer *lbv2.Service) error {
    records, err := dns.RecordSets.InZone(ctx, resource.Name("example.org."))
    if err != nil { return err }
    record, err := records.FindIdentity(ctx, "www.example.org.",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    members, err := loadbalancer.Pools.Members(ctx, resource.ID("pool-id"))
    if err != nil { return err }
    member, err := members.FindIdentity(ctx, "backend-01",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = record, member
    return nil
}
```

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    blockstoragev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindAcrossProjects(ctx context.Context, compute *computev2.Service,
    storage *blockstoragev3.Service) error {
    summary := resource.WithIdentityFindDetails(false)
    allProjects := resource.WithIdentityFindAllProjects(true)
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", summary, allProjects, strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", summary, allProjects, strict)
    if err != nil { return err }
    _, _ = server, volume
    return nil
}
```

```go
flavor, err := compute.Flavors.FindIdentity(ctx, flavorID,
    resource.WithIdentityFindFallback(resource.FindFallbackNever),
    resource.WithIdentityFindIgnoreMissing(false),
    resource.WithIdentityFindExtraSpecs(true))
```

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindFlavor(ctx context.Context, compute *computev2.Service) error {
    flavor, err := compute.Flavors.FindIdentity(ctx, "small",
        resource.WithIdentityFindExtraSpecs(true),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindQuery("minRam", "256"))
    if err != nil { return err }
    _, _ = flavor.ID, flavor.ExtraSpecs
    return nil
}
```

```go
package example

import (
    "context"

    imagev2 "github.com/JSYoo5B/go-openstacksdk/image/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindImage(ctx context.Context, images *imagev2.Service) error {
    image, err := images.Images.FindIdentity(ctx, "ubuntu",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = image.ID, image.Hidden
    return nil
}
```

```go
package example

import (
    "context"

    identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindTenantResources(ctx context.Context, identity *identityv3.Service,
    network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    domain, err := identity.Domains.FindIdentity(ctx, "Default", strict)
    if err != nil { return err }
    domainQuery := resource.WithIdentityFindQuery("domain_id", domain.ID)
    project, err := identity.Projects.FindIdentity(ctx, "app", strict, domainQuery)
    if err != nil { return err }
    user, err := identity.Users.FindIdentity(ctx, "app-user", strict, domainQuery)
    if err != nil { return err }
    group, err := identity.Groups.FindIdentity(ctx, "app-group", strict, domainQuery)
    if err != nil { return err }
    role, err := identity.Roles.FindIdentity(ctx, "reader", strict, domainQuery)
    if err != nil { return err }
    net, err := network.Networks.FindIdentity(ctx, "web-net", strict)
    if err != nil { return err }
    subnet, err := network.Subnets.FindIdentity(ctx, "web-subnet", strict,
        resource.WithIdentityFindQuery("network_id", net.ID))
    if err != nil { return err }
    _, _, _, _, _ = project, user, group, role, subnet
    return nil
}
```

```go
package example

import (
    "context"
    "net/url"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindNetworkExtensions(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    filter := resource.WithIdentityFindOptions(resource.IdentityFindOpts{
        Query: url.Values{"project_id": {projectID}, "fields": {"id", "name"}},
    })
    strict := resource.WithIdentityFindIgnoreMissing(false)
    router, err := network.Routers.FindIdentity(ctx, "app-router", filter, strict)
    if err != nil { return err }
    group, err := network.SecurityGroups.FindIdentity(ctx, "app-sg", filter, strict)
    if err != nil { return err }
    _, _ = router, group
    return nil
}
```

```go
package example

import (
    "context"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindSubnetPoolAndTrunk(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    filter := resource.WithIdentityFindQuery("project_id", projectID)
    strict := resource.WithIdentityFindIgnoreMissing(false)
    pool, err := network.SubnetPools.FindIdentity(ctx, "app-pool", filter, strict)
    if err != nil { return err }
    trunk, err := network.Trunks.FindIdentity(ctx, "app-trunk", filter, strict)
    if err != nil { return err }
    _, _ = pool.Prefixes, trunk.Subports
    return nil
}
```

```go
package example

import (
    "context"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindQoSAndAddresses(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    policy, err := network.QoSPolicies.FindIdentity(ctx, "app-qos",
        resource.WithIdentityFindQuery("shared", "true"), strict)
    if err != nil { return err }
    group, err := network.SecurityAddressGroups.FindIdentity(ctx, "app-addresses",
        resource.WithIdentityFindQuery("project_id", projectID), strict)
    if err != nil { return err }
    _, _ = policy.Rules, group.Addresses
    return nil
}
```

## References

- [Finding](../clustering/v1/finding/README.md)
- [User Read APIs](../compute/user-read-apis.md)
- [SDK Support Ledger](sdk-support-ledger.md)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439)
- [Nova find_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1023)
- [Cinder find_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L855)
- [Neutron find_port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950)
- [Designate find_recordset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py#L337)
- [Octavia find_member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py#L542)
- [find_network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214)
- [find_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)
- [find_project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1063)
- [find_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1379)
- [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L782)
- [find_domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L259)
- [find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1743)
- [Listing](../network/v2/extensions/layer3/routers/listing/README.md)
- [Listing](../network/v2/extensions/security/groups/listing/README.md)
- [find_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5570)
- [find_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598)
- [Router query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py#L36)
- [SecurityGroup query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py#L29)
- [find_subnet_pool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7621)
- [find_trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7869)
- [SubnetPool query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L34)
- [Trunk query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py#L33)
- [Listing](../network/v2/extensions/subnetpools/listing/README.md)
- [Listing](../network/v2/extensions/trunks/listing/README.md)
- [find_qos_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5130)
- [find_address_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L346)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21)
- [AddressGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/address_group.py#L22)
- [find_tap_mirror](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9096)
- [find_segment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7192)
- [find_vpn_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8592)
- [find_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6437)
- [find_bgpvpn](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1087)
- [find_bgp_peer](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L848)
- [find_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2177)
- [find_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L956)
- [find_region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1636)
- [find_aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L482)
- [get_aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1922)
- [get_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L846)
- [users.API.FindRecord](../identity/v3/users/python-record-find.md)
- [Python Record Find](../identity/v3/users/python-record-find.md)
- [Python Record Reads](../identity/v3/projects/python-record-reads.md)
- [Python Records](../identity/v3/groups/python-records.md)
- [Roles.FindRecord](../identity/v3/roles/python-record-find.md)
