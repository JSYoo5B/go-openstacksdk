# Revision Updates

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
network = conn.network.update_network(network_id, if_revision=0, name="renamed")
```

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func renameNetwork(ctx context.Context, service *networkv2.Service, id, name string, revision int) (*networks.Network, error) {
    options := networks.UpdateOpts{
        Name: &name,
        RevisionNumber: &revision,
    }
    updated, err := service.Networks.Update(ctx, id, networks.UpdateOpts{},
        networks.WithUpdateOptions(options))
    if err != nil {
        var response gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &response) {
            fmt.Printf("HTTP%d body=%q headers=%v\n",
                response.Actual, response.Body, response.ResponseHeader)
        }
    }
    return updated, err
}

func main() {}
```

## References

- [networks](networks/api_generated.go)
- [ports](ports/api_generated.go)
- [subnets](subnets/api_generated.go)
- [routers](extensions/layer3/routers/api_generated.go)
- [floatingips](extensions/layer3/floatingips/api_generated.go)
- [groups](extensions/security/groups/api_generated.go)
- [trunks](extensions/trunks/api_generated.go)
- [subnetpools](extensions/subnetpools/api_generated.go)
- [policies](extensions/qos/policies/api_generated.go)
- [pinned Network Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L213-L229)
- [NetworkResource.if_match](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py#L23-L29)
- [Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py#L22)
- [SubnetPool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L23)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21)
- [dirty/cache·no-op commit·session](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1993)
- [Python Update Record](extensions/qos/policies/python-update-record.md)
