# Network

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
network = conn.network.find_network("private", ignore_missing=False)
for network in conn.network.networks(is_router_external=True):
    print(network.id)
```

See `Example_networkFragment1` in [example_network_fragment1_test.go](example_network_fragment1_test.go), compiled by `go test`.

```go
ready, err := service.Networks.Wait(ctx, resource.ID(network.ID), "ACTIVE")
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Networks.Delete(ctx, resource.Name("private"),
    resource.WithMissingError()); err != nil {
    return err
}
```

```python
server = conn.compute.find_server("web", ignore_missing=False)
floating_ip = conn.create_floating_ip(
    network="public", server=server, nat_destination="private", wait=True)
```

```go
service, err := conn.Network(ctx)
if err != nil { return err }
created, err := service.FloatingIPs.Create(ctx,
    network.CreateFloatingIPRequest{Network: resource.Name("public")},
    network.WithServer(resource.Name("web")),
    network.WithNATDestination(resource.Name("private")),
    network.WithWait(resource.WithTimeout(time.Minute)))
if err != nil {
    if created != nil {
        fmt.Println("created floating IP remains:", created.ID)
    }
    return err
}
fmt.Println(created.FloatingIP, created.PortID, created.FixedIP)
```

```go
created, err := service.FloatingIPs.Create(ctx,
    network.CreateFloatingIPRequest{Network: resource.ID("external-network-id")},
    network.WithServer(resource.ID("server-id")),
    network.WithPort(resource.ID("port-id")),
    network.WithFixedAddress("10.0.0.10"),
    network.WithDescription("web ingress"),
    request.Field("vendor_enabled", false))
```

## References

- [..](../README.md)
- [Floating Ip Attach](floating-ip-attach.md)
- [Floating Ip Ensure](floating-ip-ensure.md)
- [Compute·Connection Add helper](../compute/server-ip-helpers.md)
- [Compute IP dispatch](../compute/server-ip-dispatch.md)
- [Network](https://docs.openstack.org/openstacksdk/latest/user/proxies/network.html)
- [Finding Identities](../docs/finding-identities.md)
- [Listing](../docs/listing.md)
- [Subnets](v2/subnets/README.md)
- [Listing](v2/extensions/security/addressgroups/listing/README.md)
- [Listing](v2/extensions/qos/policies/listing/README.md)
- [Listing](v2/extensions/subnetpools/listing/README.md)
- [Listing](v2/networks/listing/README.md)
- [Listing](v2/extensions/layer3/routers/listing/README.md)
- [Listing](v2/extensions/security/groups/listing/README.md)
- [Listing](v2/extensions/trunks/listing/README.md)
- [Network Mutations](network-mutations.md)
- [Network v2 API](v2/README.md)
- [FloatingIPs.Ensure](floating-ip-ensure.md)
- [FloatingIPs.Attach](floating-ip-attach.md)
- [network_test.go](network_test.go)
- [floating_ip_test.go](floating_ip_test.go)
- [Collections Test](../connectiontest/collections_test.go)
- [PrepareEnsure·EnsurePrepared](floating-ip-plan.md)
- [Server Automatic Ip](../compute/server-automatic-ip.md)
- [CreateWithFloatingIP](../compute/create-with-floating-ip.md)
- [Network Roles](network-roles.md)
- [Server Addresses](../compute/server-addresses.md)
- [Server Nova Floating Ip](../compute/server-nova-floating-ip.md)
- [FloatingIPs.Available](floating-ip-available.md)
- [Connection·Compute Available](../compute/floating-ip-available.md)
- [Floating Ip Queries](../compute/floating-ip-queries.md)
- [Floating Ip Delete](../compute/floating-ip-delete.md)
- [Floating Ip Create](../compute/floating-ip-create.md)
- [Floating Ip Unattached Delete](../compute/floating-ip-unattached-delete.md)
- [Cloud Firewall Policy Insert Rule Python](cloud-firewall-policy-insert-rule-python.md)
- [Cloud Firewall Policy Remove Rule Python](cloud-firewall-policy-remove-rule-python.md)
- [Python Available](v2/extensions/layer3/floatingips/python-available.md)
- [IP availability records](v2/extensions/networkipavailabilities/python-records.md)
