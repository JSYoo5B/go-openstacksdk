# Listing

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListServers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	for server, err := range service.Servers.Resources.List(ctx,
		resource.WithName("worker.*"),
		resource.WithStatus("ACTIVE"),
		resource.WithPageSize(50),
		resource.WithMaxItems(100),
	) {
		if err != nil {
			return err
		}
		fmt.Println(server.ID, server.Name, server.Status)
	}
	return nil
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListRecordSets(ctx context.Context, conn *sdk.Connection, zoneID string) error {
	service, err := conn.DNS(ctx)
	if err != nil {
		return err
	}
	scope, err := service.RecordSets.InZone(ctx, resource.ID(zoneID))
	if err != nil {
		return err
	}
	records, err := scope.All(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	for _, record := range records {
		fmt.Println(record.ID, record.Name, record.Type, record.Records)
	}
	return nil
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListMembers(ctx context.Context, conn *sdk.Connection, poolID string) error {
	service, err := conn.LoadBalancer(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Pools.Members(ctx, resource.ID(poolID))
	if err != nil {
		return err
	}
	for member, err := range scope.List(ctx,
		resource.WithMaxItems(10),
		resource.WithPaginated(false),
		resource.WithStatus("ACTIVE"),
	) {
		if err != nil {
			return err
		}
		fmt.Println(member.ID, member.Name, member.ProvisioningStatus)
	}
	return nil
}
```

```go
package examples

import (
    "context"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListLocalMatches(ctx context.Context, service *networkv2.Service) error {
    policies, err := service.QoSPolicies.Resources.All(ctx,
        resource.WithBodyFilter("rules", []map[string]any{}),
        resource.WithMaxItems(100))
    if err != nil { return err }
    groups, err := service.SecurityAddressGroups.Resources.All(ctx,
        resource.WithBodyFilters(map[string]any{
            "addresses": []string{"192.0.2.0/24"},
        }), resource.WithPaginated(false))
    if err != nil { return err }
    _, _ = policies, groups
    return nil
}

func ListSubnetMatches(ctx context.Context, service *networkv2.Service) error {
    for value, err := range service.Networks.Resources.List(ctx,
        resource.WithBodyFilter("subnet_ids", []string{"subnet-a", "subnet-b"}),
        resource.WithStatus("ACTIVE")) {
        if err != nil { return err }
        _ = value
    }
    _, err := service.SubnetPools.Resources.All(ctx,
        resource.WithBodyFilter("prefixes", []string{"192.0.2.0/24"}),
        resource.WithMaxItems(100))
    return err
}
```

## References

- [Manila access rule scope](../sharedfilesystems/v2/shareaccessrules/README.md)
- [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2355)
- [containers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L143-L152)
- [objects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L270)
- [Nova action](../compute/v2/instanceactions/README.md)
- [Swift container](../objectstorage/v1/containers/README.md)
- [Swift object](../objectstorage/v1/objects/README.md)
- [Trove database](../db/v1/databases/README.md)
- [Trove user](../db/v1/users/README.md)
- [Generated List Controls Test](../api/generated_list_controls_test.go)
- [Native Scope List Controls Test](../api/native_scope_list_controls_test.go)
- [List Control Test](../resource/list_control_test.go)
- [Stream Test](../resource/stream_test.go)
- [Listing](../network/v2/networks/listing/README.md)
- [Network List Filters Test](../api/network_list_filters_test.go)
- [Connection Network Filters Test](../connectiontest/connection_network_filters_test.go)
- [Boolean Test](../internal/jsonfilter/boolean_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/network.json)
- [Network Filters Test](../internal/cmd/sdkgen/network_filters_test.go)
- [Listing](../network/v2/extensions/layer3/routers/listing/README.md)
- [Router List Filters Test](../api/router_list_filters_test.go)
- [Connection Router Filters Test](../connectiontest/connection_router_filters_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/router.json)
- [Router Filters Test](../internal/cmd/sdkgen/router_filters_test.go)
- [Listing](../network/v2/extensions/security/groups/listing/README.md)
- [Security Group List Filters Test](../api/security_group_list_filters_test.go)
- [Connection Security Group Filters Test](../connectiontest/connection_security_group_filters_test.go)
- [Security Group Bodies Test](../internal/nativefind/security_group_bodies_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/security_group.json)
- [Security Group Filters Test](../internal/cmd/sdkgen/security_group_filters_test.go)
- [Listing](../network/v2/extensions/trunks/listing/README.md)
- [Trunk List Filters Test](../api/trunk_list_filters_test.go)
- [Connection Trunk Filters Test](../connectiontest/connection_trunk_filters_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/trunk.json)
- [Trunk Filters Test](../internal/cmd/sdkgen/trunk_filters_test.go)
- [Subnets](../network/v2/subnets/README.md)
- [Network Subnet Semantic Filters Test](../api/network_subnet_semantic_filters_test.go)
- [Filters Test](../resource/filters_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/subnet.json)
- [SDKgen](../internal/cmd/sdkgen/README.md)
- [Listing](../network/v2/extensions/subnetpools/listing/README.md)
- [Subnet Pool List Filters Test](../api/subnet_pool_list_filters_test.go)
- [Connection Subnet Pool Filters Test](../connectiontest/connection_subnet_pool_filters_test.go)
- [AST manifest](../api/openstacksdk/resources/network/v2/subnet_pool.json)
- [Listing](../network/v2/extensions/security/addressgroups/listing/README.md)
- [AST manifest](../api/openstacksdk/resources/network/v2/address_group.json)
- [Listing](../network/v2/extensions/qos/policies/listing/README.md)
- [AST manifest](../api/openstacksdk/resources/network/v2/qos_policy.json)
- [Listing](../keymanager/v1/secrets/listing/README.md)
- [Secret AST manifest](../api/openstacksdk/resources/key_manager/v1/secret.json)
- [Listing](../keymanager/v1/containers/listing/README.md)
- [Container AST manifest](../api/openstacksdk/resources/key_manager/v1/container.json)
- [Listing](../keymanager/v1/orders/listing/README.md)
- [Order AST manifest](../api/openstacksdk/resources/key_manager/v1/order.json)
- [Subnet README](../network/v2/subnets/README.md)
- [qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5177)
- [address_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L392)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217)
- [subnet_ids](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L119)
- [prefixes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L91)
- [Network Body Filters Test](../api/network_body_filters_test.go)
- [Network Subnet Body Filters Test](../api/network_subnet_body_filters_test.go)
- [Body Filters Test](../resource/body_filters_test.go)
- [Filter Test](../internal/jsonfilter/filter_test.go)
- [Network Subnet Raw Body Filters Test](../api/network_subnet_raw_body_filters_test.go)
- [Body Record Test](../resource/body_record_test.go)
- [Integer Test](../internal/jsonfilter/integer_test.go)

## Explicit native Body filters

`resource.WithBodyFilter` and `WithBodyFilters` compare explicitly supported raw response fields locally. QoS policies expose rules, address groups expose addresses, subnet pools expose prefixes, and networks expose subnets. Network also accepts subnet_ids as an alias, rejecting both aliases together. Options snapshot JSON and compare whole arrays; nil and empty arrays differ. Raw caps apply before local filtering, and whole-page native decoding can still fail beyond a cap. These options belong to Resources.List/All, not native typed List or FindIdentity.
