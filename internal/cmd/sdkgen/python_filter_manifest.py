#!/usr/bin/env python3
"""Extract audited resource filters without importing OpenStack.

Only Python's standard-library AST is used. The source checkout is data, never
executed; unexpected expression shapes fail rather than becoming guessed fields.
AST hashes are CPython-minor-version dependent, as recorded in the manifest.
"""

import argparse
import ast
import hashlib
import json
import pathlib
import sys


PIN = "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe"
FILES = (
    "openstack/network/v2/subnet.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
RESOURCE = "openstack.network.v2.subnet.Subnet"
ANCHORS = (
    ("openstack/network/v2/subnet.py", "Subnet"),
    ("openstack/network/v2/subnet.py", "Subnet._query_mapping"),
    ("openstack/common/tag.py", "TagMixin._tag_query_parameters"),
    ("openstack/common/tag.py", "TagMixin.tags"),
    ("openstack/network/v2/_base.py", "NetworkResource.revision_number"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/network/v2/_proxy.py", "Proxy.subnets"),
)
SECRET_RESOURCE = "openstack.key_manager.v1.secret.Secret"
SECRET_FILES = (
    "openstack/key_manager/v1/secret.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/key_manager/v1/_proxy.py",
    "openstack/key_manager/v1/_format.py",
    "openstack/format.py",
)
SECRET_ANCHORS = (
    ("openstack/key_manager/v1/secret.py", "Secret"),
    ("openstack/key_manager/v1/secret.py", "Secret._query_mapping"),
    ("openstack/key_manager/v1/secret.py", "Secret.secret_id"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._get_id"),
    ("openstack/resource.py", "Resource.__init__"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource._attr_to_dict"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/key_manager/v1/_proxy.py", "Proxy.secrets"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID.deserialize"),
    ("openstack/format.py", "Formatter"),
)
CONTAINER_RESOURCE = "openstack.key_manager.v1.container.Container"
CONTAINER_FILES = (
    "openstack/key_manager/v1/container.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/key_manager/v1/_proxy.py",
    "openstack/key_manager/v1/_format.py",
    "openstack/format.py",
)
CONTAINER_ANCHORS = (
    ("openstack/key_manager/v1/container.py", "Container"),
    # Container has no own query mapping: prove the effective inherited
    # QueryParameters() declaration and its limit/marker defaults separately.
    ("openstack/resource.py", "Resource._query_mapping"),
    ("openstack/key_manager/v1/container.py", "Container.container_id"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._get_id"),
    ("openstack/resource.py", "Resource.__init__"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource._attr_to_dict"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/key_manager/v1/_proxy.py", "Proxy.containers"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID.deserialize"),
    ("openstack/format.py", "Formatter"),
)
ORDER_RESOURCE = "openstack.key_manager.v1.order.Order"
ORDER_FILES = (
    "openstack/key_manager/v1/order.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/key_manager/v1/_proxy.py",
    "openstack/key_manager/v1/_format.py",
    "openstack/format.py",
)
ORDER_ANCHORS = (
    ("openstack/key_manager/v1/order.py", "Order"),
    ("openstack/resource.py", "Resource._query_mapping"),
    ("openstack/key_manager/v1/order.py", "Order.order_id"),
    ("openstack/key_manager/v1/order.py", "Order.secret_id"),
    ("openstack/key_manager/v1/order.py", "Order.meta"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._get_id"),
    ("openstack/resource.py", "Resource.__init__"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource._attr_to_dict"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/key_manager/v1/_proxy.py", "Proxy.orders"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID.deserialize"),
    ("openstack/format.py", "Formatter"),
)
ADDRESS_GROUP_RESOURCE = "openstack.network.v2.address_group.AddressGroup"
ADDRESS_GROUP_FILES = (
    "openstack/network/v2/address_group.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
ADDRESS_GROUP_ANCHORS = (
    ("openstack/network/v2/address_group.py", "AddressGroup"),
    ("openstack/network/v2/address_group.py", "AddressGroup._query_mapping"),
    # Descriptor aliases do not add query keys. Keep the deprecated local
    # tenant_id property separate from the queried project_id property.
    ("openstack/network/v2/address_group.py", "AddressGroup.project_id"),
    ("openstack/network/v2/address_group.py", "AddressGroup.tenant_id"),
    ("openstack/network/v2/address_group.py", "AddressGroup.addresses"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/network/v2/_proxy.py", "Proxy.address_groups"),
)
QOS_POLICY_RESOURCE = "openstack.network.v2.qos_policy.QoSPolicy"
QOS_POLICY_FILES = (
    "openstack/network/v2/qos_policy.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
QOS_POLICY_ANCHORS = (
    ("openstack/network/v2/qos_policy.py", "QoSPolicy"),
    ("openstack/network/v2/qos_policy.py", "QoSPolicy._query_mapping"),
    # Query aliases come from QueryParameters, independently of the
    # project_id response fallback and deprecated tenant_id local property.
    ("openstack/network/v2/qos_policy.py", "QoSPolicy.project_id"),
    ("openstack/network/v2/qos_policy.py", "QoSPolicy.tenant_id"),
    ("openstack/network/v2/qos_policy.py", "QoSPolicy.rules"),
    ("openstack/common/tag.py", "TagMixin._tag_query_parameters"),
    ("openstack/common/tag.py", "TagMixin.tags"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/network/v2/_proxy.py", "Proxy.qos_policies"),
)
SUBNET_POOL_RESOURCE = "openstack.network.v2.subnet_pool.SubnetPool"
SUBNET_POOL_FILES = (
    "openstack/network/v2/subnet_pool.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
SUBNET_POOL_ANCHORS = (
    ("openstack/network/v2/subnet_pool.py", "SubnetPool"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool._query_mapping"),
    # Response aliases and local integer/list descriptors are independent of
    # QueryParameters and its tag expansion; keep every coercion explicit.
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.project_id"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.tenant_id"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.prefixes"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.default_prefix_length"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.default_quota"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.maximum_prefix_length"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.minimum_prefix_length"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.revision_number"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.created_at"),
    ("openstack/network/v2/subnet_pool.py", "SubnetPool.updated_at"),
    ("openstack/network/v2/_base.py", "TagMixinNetwork"),
    ("openstack/common/tag.py", "TagMixin._tag_query_parameters"),
    ("openstack/common/tag.py", "TagMixin.tags"),
    ("openstack/resource.py", "Body"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._collect_attrs"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/resource.py", "Resource.list._dict_filter"),
    ("openstack/fields.py", "_BaseComponent.__init__"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/network/v2/_proxy.py", "Proxy.subnet_pools"),
)
NETWORK_RESOURCE = "openstack.network.v2.network.Network"
NETWORK_FILES = (
    "openstack/network/v2/network.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
# Full classification/default/accessor anchors include inherited revision,
# bool None handling and query-mapped False defaults. Hash coverage of a
# continuation branch does not claim its runtime implementation.
NETWORK_ANCHORS = (
    ('openstack/network/v2/network.py', 'Network'),
    ('openstack/network/v2/network.py', 'Network._query_mapping'),
    ('openstack/network/v2/network.py', 'Network.availability_zone_hints'),
    ('openstack/network/v2/network.py', 'Network.availability_zones'),
    ('openstack/network/v2/network.py', 'Network.created_at'),
    ('openstack/network/v2/network.py', 'Network.dns_domain'),
    ('openstack/network/v2/network.py', 'Network.is_default'),
    ('openstack/network/v2/network.py', 'Network.mtu'),
    ('openstack/network/v2/network.py', 'Network.pvlan'),
    ('openstack/network/v2/network.py', 'Network.qos_policy_id'),
    ('openstack/network/v2/network.py', 'Network.segments'),
    ('openstack/network/v2/network.py', 'Network.subnet_ids'),
    ('openstack/network/v2/network.py', 'Network.updated_at'),
    ('openstack/network/v2/network.py', 'Network.is_vlan_transparent'),
    ('openstack/network/v2/network.py', 'Network.is_vlan_qinq'),
    ('openstack/resource.py', 'Resource'),
    ('openstack/resource.py', 'Resource.id'),
    ('openstack/resource.py', 'Resource.name'),
    ('openstack/resource.py', 'Resource._max_microversion'),
    ('openstack/resource.py', 'Resource.__init__'),
    ('openstack/resource.py', 'Resource._attributes_iterator'),
    ('openstack/resource.py', 'Resource._collect_attrs'),
    ('openstack/resource.py', 'Resource.__getattribute__'),
    ('openstack/resource.py', 'Resource.__getitem__'),
    ('openstack/resource.py', 'Resource._alternate_id'),
    ('openstack/resource.py', 'Resource.to_dict'),
    ('openstack/resource.py', 'Resource.list'),
    ('openstack/resource.py', 'Resource._get_next_link'),
    ('openstack/resource.py', 'QueryParameters.__init__'),
    ('openstack/resource.py', 'QueryParameters._validate'),
    ('openstack/resource.py', 'QueryParameters._transpose'),
    ('openstack/fields.py', '_BaseComponent.__init__'),
    ('openstack/fields.py', '_BaseComponent.__get__'),
    ('openstack/fields.py', '_convert_type'),
    ('openstack/proxy.py', 'Proxy._list'),
    ('openstack/network/v2/_proxy.py', 'Proxy.networks'),
    ('openstack/common/tag.py', 'TagMixin'),
    ('openstack/common/tag.py', 'TagMixin._tag_query_parameters'),
    ('openstack/common/tag.py', 'TagMixin.tags'),
    ('openstack/network/v2/_base.py', 'NetworkResource'),
    ('openstack/network/v2/_base.py', 'NetworkResource.revision_number'),
    ('openstack/network/v2/_base.py', 'TagMixinNetwork'),
    ('openstack/resource.py', 'ResourceMixinProtocol'),
)
ROUTER_RESOURCE = "openstack.network.v2.router.Router"
ROUTER_FILES = (
    "openstack/network/v2/router.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
# Router overrides inherited revision_number with raw revision. Project's
# response alias does not widen QueryParameters to local tenant_id. Include
# separate own descriptors plus inherited/default/accessor source evidence.
ROUTER_ANCHORS = (
    ('openstack/network/v2/router.py', 'Router'),
    ('openstack/network/v2/router.py', 'Router._query_mapping'),
    ('openstack/network/v2/router.py', 'Router.availability_zone_hints'),
    ('openstack/network/v2/router.py', 'Router.availability_zones'),
    ('openstack/network/v2/router.py', 'Router.created_at'),
    ('openstack/network/v2/router.py', 'Router.enable_ndp_proxy'),
    ('openstack/network/v2/router.py', 'Router.evpn_vni'),
    ('openstack/network/v2/router.py', 'Router.external_gateway_info'),
    ('openstack/network/v2/router.py', 'Router.project_id'),
    ('openstack/network/v2/router.py', 'Router.tenant_id'),
    ('openstack/network/v2/router.py', 'Router.revision_number'),
    ('openstack/network/v2/router.py', 'Router.routes'),
    ('openstack/network/v2/router.py', 'Router.updated_at'),
    ('openstack/resource.py', 'Resource'),
    ('openstack/resource.py', 'Resource.id'),
    ('openstack/resource.py', 'Resource.name'),
    ('openstack/resource.py', 'Resource._max_microversion'),
    ('openstack/resource.py', 'Resource.__init__'),
    ('openstack/resource.py', 'Resource._attributes_iterator'),
    ('openstack/resource.py', 'Resource._collect_attrs'),
    ('openstack/resource.py', 'Resource.__getattribute__'),
    ('openstack/resource.py', 'Resource.__getitem__'),
    ('openstack/resource.py', 'Resource._alternate_id'),
    ('openstack/resource.py', 'Resource.to_dict'),
    ('openstack/resource.py', 'Resource.list'),
    ('openstack/resource.py', 'Resource._get_next_link'),
    ('openstack/resource.py', 'QueryParameters.__init__'),
    ('openstack/resource.py', 'QueryParameters._validate'),
    ('openstack/resource.py', 'QueryParameters._transpose'),
    ('openstack/fields.py', '_BaseComponent.__init__'),
    ('openstack/fields.py', '_BaseComponent.__get__'),
    ('openstack/fields.py', '_convert_type'),
    ('openstack/proxy.py', 'Proxy._list'),
    ('openstack/network/v2/_proxy.py', 'Proxy.routers'),
    ('openstack/common/tag.py', 'TagMixin'),
    ('openstack/common/tag.py', 'TagMixin._tag_query_parameters'),
    ('openstack/common/tag.py', 'TagMixin.tags'),
    ('openstack/network/v2/_base.py', 'NetworkResource'),
    ('openstack/network/v2/_base.py', 'NetworkResource.revision_number'),
    ('openstack/network/v2/_base.py', 'TagMixinNetwork'),
    ('openstack/resource.py', 'ResourceMixinProtocol'),
)
SECURITY_GROUP_RESOURCE = "openstack.network.v2.security_group.SecurityGroup"
SECURITY_GROUP_FILES = (
    "openstack/network/v2/security_group.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
# Both tenant and project are explicit query attributes. The response alias
# does not collapse them, and inherited revision_number stays query-mapped.
SECURITY_GROUP_ANCHORS = (
    ('openstack/network/v2/security_group.py', 'SecurityGroup'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup._query_mapping'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.created_at'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.description'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.name'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.stateful'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.project_id'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.security_group_rules'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.tenant_id'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.updated_at'),
    ('openstack/network/v2/security_group.py', 'SecurityGroup.is_shared'),
    ('openstack/resource.py', 'Resource'),
    ('openstack/resource.py', 'Resource.id'),
    ('openstack/resource.py', 'Resource.name'),
    ('openstack/resource.py', 'Resource._max_microversion'),
    ('openstack/resource.py', 'Resource.__init__'),
    ('openstack/resource.py', 'Resource._attributes_iterator'),
    ('openstack/resource.py', 'Resource._collect_attrs'),
    ('openstack/resource.py', 'Resource.__getattribute__'),
    ('openstack/resource.py', 'Resource.__getitem__'),
    ('openstack/resource.py', 'Resource._alternate_id'),
    ('openstack/resource.py', 'Resource.to_dict'),
    ('openstack/resource.py', 'Resource.list'),
    ('openstack/resource.py', 'Resource._get_next_link'),
    ('openstack/resource.py', 'QueryParameters.__init__'),
    ('openstack/resource.py', 'QueryParameters._validate'),
    ('openstack/resource.py', 'QueryParameters._transpose'),
    ('openstack/fields.py', '_BaseComponent.__init__'),
    ('openstack/fields.py', '_BaseComponent.__get__'),
    ('openstack/fields.py', '_convert_type'),
    ('openstack/proxy.py', 'Proxy._list'),
    ('openstack/network/v2/_proxy.py', 'Proxy.security_groups'),
    ('openstack/common/tag.py', 'TagMixin'),
    ('openstack/common/tag.py', 'TagMixin._tag_query_parameters'),
    ('openstack/common/tag.py', 'TagMixin.tags'),
    ('openstack/network/v2/_base.py', 'NetworkResource'),
    ('openstack/network/v2/_base.py', 'NetworkResource.revision_number'),
    ('openstack/network/v2/_base.py', 'TagMixinNetwork'),
    ('openstack/resource.py', 'ResourceMixinProtocol'),
)
TRUNK_RESOURCE = "openstack.network.v2.trunk.Trunk"
TRUNK_FILES = (
    "openstack/network/v2/trunk.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
# Trunk uses Resource + TagMixin, not NetworkResource. Native timestamps and
# revision_number must not become semantic descriptors. Sub_ports is queried.
TRUNK_ANCHORS = (
    ('openstack/network/v2/trunk.py', 'Trunk'),
    ('openstack/network/v2/trunk.py', 'Trunk._query_mapping'),
    ('openstack/network/v2/trunk.py', 'Trunk.name'),
    ('openstack/network/v2/trunk.py', 'Trunk.project_id'),
    ('openstack/network/v2/trunk.py', 'Trunk.tenant_id'),
    ('openstack/network/v2/trunk.py', 'Trunk.description'),
    ('openstack/network/v2/trunk.py', 'Trunk.is_admin_state_up'),
    ('openstack/network/v2/trunk.py', 'Trunk.port_id'),
    ('openstack/network/v2/trunk.py', 'Trunk.status'),
    ('openstack/network/v2/trunk.py', 'Trunk.sub_ports'),
    ('openstack/resource.py', 'Resource'),
    ('openstack/resource.py', 'Resource.id'),
    ('openstack/resource.py', 'Resource.name'),
    ('openstack/resource.py', 'Resource._max_microversion'),
    ('openstack/resource.py', 'Resource.__init__'),
    ('openstack/resource.py', 'Resource._attributes_iterator'),
    ('openstack/resource.py', 'Resource._collect_attrs'),
    ('openstack/resource.py', 'Resource.__getattribute__'),
    ('openstack/resource.py', 'Resource.__getitem__'),
    ('openstack/resource.py', 'Resource._alternate_id'),
    ('openstack/resource.py', 'Resource.to_dict'),
    ('openstack/resource.py', 'Resource.list'),
    ('openstack/resource.py', 'Resource._get_next_link'),
    ('openstack/resource.py', 'QueryParameters.__init__'),
    ('openstack/resource.py', 'QueryParameters._validate'),
    ('openstack/resource.py', 'QueryParameters._transpose'),
    ('openstack/fields.py', '_BaseComponent.__init__'),
    ('openstack/fields.py', '_BaseComponent.__get__'),
    ('openstack/fields.py', '_convert_type'),
    ('openstack/proxy.py', 'Proxy._list'),
    ('openstack/network/v2/_proxy.py', 'Proxy.trunks'),
    ('openstack/common/tag.py', 'TagMixin'),
    ('openstack/common/tag.py', 'TagMixin._tag_query_parameters'),
    ('openstack/common/tag.py', 'TagMixin.tags'),
    ('openstack/resource.py', 'ResourceMixinProtocol'),
)
TARGETS = {
    "subnet": (RESOURCE, FILES, ANCHORS, "gophercloudsdk/network/v2/subnets"),
    "trunk": (TRUNK_RESOURCE, TRUNK_FILES, TRUNK_ANCHORS,
              "gophercloudsdk/network/v2/extensions/trunks"),
    "network": (NETWORK_RESOURCE, NETWORK_FILES, NETWORK_ANCHORS,
                "gophercloudsdk/network/v2/networks"),
    "router": (ROUTER_RESOURCE, ROUTER_FILES, ROUTER_ANCHORS,
               "gophercloudsdk/network/v2/extensions/layer3/routers"),
    "security_group": (SECURITY_GROUP_RESOURCE, SECURITY_GROUP_FILES,
                       SECURITY_GROUP_ANCHORS,
                       "gophercloudsdk/network/v2/extensions/security/groups"),
    "secret": (SECRET_RESOURCE, SECRET_FILES, SECRET_ANCHORS,
               "gophercloudsdk/keymanager/v1/secrets"),
    "container": (CONTAINER_RESOURCE, CONTAINER_FILES, CONTAINER_ANCHORS,
                  "gophercloudsdk/keymanager/v1/containers"),
    "order": (ORDER_RESOURCE, ORDER_FILES, ORDER_ANCHORS,
              "gophercloudsdk/keymanager/v1/orders"),
    "address_group": (ADDRESS_GROUP_RESOURCE, ADDRESS_GROUP_FILES,
                      ADDRESS_GROUP_ANCHORS,
                      "gophercloudsdk/network/v2/extensions/security/addressgroups"),
    "qos_policy": (QOS_POLICY_RESOURCE, QOS_POLICY_FILES, QOS_POLICY_ANCHORS,
                   "gophercloudsdk/network/v2/extensions/qos/policies"),
    "subnet_pool": (SUBNET_POOL_RESOURCE, SUBNET_POOL_FILES, SUBNET_POOL_ANCHORS,
                    "gophercloudsdk/network/v2/extensions/subnetpools"),
}


def assignment(node):
    if isinstance(node, ast.Assign) and len(node.targets) == 1:
        if isinstance(node.targets[0], ast.Name):
            return node.targets[0].id, node.value
    if isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
        return node.target.id, node.value
    return None


class Source:
    def __init__(self, root, files=FILES):
        self.raw = {}
        self.trees = {}
        self.modules = {}
        self.imports = {}
        self.classes = {}
        self.attributes = {}
        for path in files:
            raw = (root / path).read_bytes()
            module = path[:-3].replace("/", ".")
            tree = ast.parse(raw, filename=path)
            self.raw[path] = raw
            self.trees[path] = tree
            self.modules[module] = tree
            aliases = {}
            for node in tree.body:
                if isinstance(node, ast.ImportFrom) and node.level == 0:
                    for alias in node.names:
                        aliases[alias.asname or alias.name] = (
                            node.module + "." + alias.name
                        )
                elif isinstance(node, ast.Import):
                    for alias in node.names:
                        aliases[alias.asname or alias.name.split(".")[0]] = (
                            alias.name if alias.asname else alias.name.split(".")[0]
                        )
                elif isinstance(node, ast.ClassDef):
                    qualified = module + "." + node.name
                    self.classes[qualified] = node
                    attrs = {}
                    for member in node.body:
                        pair = assignment(member)
                        if pair:
                            attrs[pair[0]] = (module, pair[1], member)
                        elif isinstance(member, (ast.FunctionDef, ast.AsyncFunctionDef)):
                            attrs[member.name] = (module, member, member)
                    self.attributes[qualified] = attrs
            self.imports[module] = aliases

    def resolve(self, module, node):
        if isinstance(node, ast.Subscript):
            return self.resolve(module, node.value)
        if isinstance(node, ast.Name):
            if node.id in self.imports[module]:
                result = self.imports[module][node.id]
            elif node.id in {"dict", "list", "str", "int", "bool", "object"}:
                result = "builtins." + node.id
            else:
                result = module + "." + node.id
        elif isinstance(node, ast.Attribute):
            result = self.resolve(module, node.value) + "." + node.attr
        else:
            raise ValueError("unsupported reference: " + ast.dump(node))
        parent, _, name = result.rpartition(".")
        if parent in self.imports and name in self.imports[parent]:
            return self.imports[parent][name]
        return result

    def bases(self, qualified):
        if qualified not in self.classes:
            if qualified not in {"builtins.dict", "typing.Protocol"}:
                raise ValueError("unreviewed class base: " + qualified)
            return []
        module = qualified.rpartition(".")[0]
        return [self.resolve(module, base) for base in self.classes[qualified].bases]

    def mro(self, qualified):
        bases = self.bases(qualified)
        result = [qualified]
        sequences = [self.mro(base) for base in bases] + [bases[:]]
        while any(sequences):
            sequences = [sequence for sequence in sequences if sequence]
            candidate = next(
                (sequence[0] for sequence in sequences
                 if all(sequence[0] not in other[1:] for other in sequences)),
                None,
            )
            if candidate is None:
                raise ValueError("inconsistent class MRO")
            result.append(candidate)
            for sequence in sequences:
                if sequence[0] == candidate:
                    sequence.pop(0)
        return result

    def effective_attributes(self, qualified):
        attrs = {}
        for base in reversed(self.mro(qualified)):
            attrs.update(self.attributes.get(base, {}))
        return attrs

    def literal(self, module, node):
        if isinstance(node, (ast.Name, ast.Attribute)):
            reference = self.resolve(module, node)
            parent, _, member = reference.rpartition(".")
            if parent in self.classes:
                mod, value, _ = self.effective_attributes(parent)[member]
                return self.literal(mod, value)
        return ast.literal_eval(node)

    def anchor(self, path, symbol):
        nodes = self.trees[path].body
        for name in symbol.split("."):
            found = []
            for node in nodes:
                pair = assignment(node)
                if getattr(node, "name", None) == name or (pair and pair[0] == name):
                    if isinstance(node, ast.FunctionDef) and any(
                        isinstance(decorator, ast.Name) and decorator.id == "overload"
                        for decorator in node.decorator_list
                    ):
                        continue
                    found.append(node)
            if len(found) != 1:
                raise ValueError("missing/ambiguous source anchor: " + symbol)
            node = found[0]
            nodes = getattr(node, "body", [])
        return node


def control_arguments(node, excluded):
    args = node.args.posonlyargs + node.args.args + node.args.kwonlyargs
    return [arg.arg for arg in args if arg.arg not in excluded]


def implementation_policies(source):
    listing = source.anchor("openstack/resource.py", "Resource.list")
    validates = [node for node in ast.walk(listing) if isinstance(node, ast.Call)
                 and isinstance(node.func, ast.Attribute) and node.func.attr == "_validate"]
    if len(validates) != 1:
        raise ValueError("unsupported list query validation")
    unknown = [kw.value for kw in validates[0].keywords if kw.arg == "allow_unknown_params"]
    if len(unknown) != 1 or not isinstance(unknown[0], ast.Constant) or unknown[0].value is not True:
        raise ValueError("unsupported unknown filter policy")
    transposing = source.anchor("openstack/resource.py", "QueryParameters._transpose")
    def membership(node, key):
        return (isinstance(node, ast.Compare) and isinstance(node.left, ast.Name)
                and node.left.id == key and len(node.ops) == 1
                and isinstance(node.ops[0], ast.In) and len(node.comparators) == 1
                and isinstance(node.comparators[0], ast.Name) and node.comparators[0].id == "query")
    def selects(nodes, key):
        if len(nodes) != 1 or not isinstance(nodes[0], ast.Assign):
            return False
        value = nodes[0].value
        return (isinstance(value, ast.Subscript) and isinstance(value.value, ast.Name)
                and value.value.id == "query" and isinstance(value.slice, ast.Name)
                and value.slice.id == key)
    canonical = [node for node in ast.walk(transposing) if isinstance(node, ast.If)
                 and membership(node.test, "client_side")]
    if len(canonical) != 1 or not selects(canonical[0].body, "client_side"):
        raise ValueError("unsupported canonical query precedence")
    fallback = canonical[0].orelse
    if (len(fallback) != 1 or not isinstance(fallback[0], ast.If)
            or not membership(fallback[0].test, "name") or not selects(fallback[0].body, "name")):
        raise ValueError("unsupported wire query precedence")
    return "discard", "canonical_client_name_wins"


def keymanager_body_accessors(source, attrs, body, label, alternate_id, ref):
    """Keep Resource.id separate from the formatted alternate-ID descriptor."""
    if body.get("id") != {"field": "id", "response_type": None}:
        raise ValueError("unsupported " + label + " literal ID descriptor")
    module, descriptor, _ = attrs[alternate_id]
    alternate = [keyword.value for keyword in descriptor.keywords
                 if keyword.arg == "alternate_id"]
    if len(alternate) != 1 or source.literal(module, alternate[0]) is not True:
        raise ValueError("unsupported " + label + " alternate ID descriptor")
    formatter = "openstack.key_manager.v1._format.HREFToUUID"
    if body.get(alternate_id) != {"field": ref, "response_type": formatter}:
        raise ValueError("unsupported " + label + " alternate ID formatter")
    if body.get(ref) != {"field": ref, "response_type": None}:
        raise ValueError("unsupported " + label + " reference descriptor")
    formatter_class = source.classes[formatter]
    if (len(formatter_class.bases) != 1
            or source.resolve(formatter.rpartition(".")[0], formatter_class.bases[0])
            != "openstack.format.Formatter"):
        raise ValueError("unsupported " + label + " formatter base")
    # Resource.__getattribute__ reads the stored literal ID before consulting
    # the alternate wire field. It does not apply HREFToUUID to that fallback.
    # Separate AST anchors prove both access paths and the to_dict projection.
    body["id"]["response_accessor"] = "resource_id"
    body[alternate_id]["formatter"] = formatter


def order_body_accessors(source, attrs, body):
    """Prove two separate formatted references and the sole alternate ID."""
    keymanager_body_accessors(source, attrs, body, "Order", "order_id", "order_ref")
    formatter = "openstack.key_manager.v1._format.HREFToUUID"
    if body.get("secret_id") != {"field": "secret_ref", "response_type": formatter}:
        raise ValueError("unsupported Order secret ID formatter")
    if body.get("secret_ref") != {"field": "secret_ref", "response_type": None}:
        raise ValueError("unsupported Order secret reference descriptor")
    if body.get("meta") != {"field": "meta", "response_type": "dict"}:
        raise ValueError("unsupported Order metadata dict descriptor")
    for name, expected in (("order_id", {"alternate_id", "type"}),
                           ("secret_id", {"type"}), ("meta", {"type"})):
        _, descriptor, _ = attrs[name]
        keywords = [keyword.arg for keyword in descriptor.keywords]
        if len(keywords) != len(expected) or set(keywords) != expected:
            raise ValueError("unsupported Order " + name + " descriptor options")
    # secret_id uses its own secret_ref formatter but is not the Resource ID.
    # Keep both properties separate from the literal/full order_ref accessor.
    body["secret_id"]["formatter"] = formatter


def network_body_descriptors(source, attrs, body):
    """Prove exact declared response defaults without widening query names."""
    expected = {
        "availability_zone_hints": ("availability_zone_hints", "list", None),
        "availability_zones": ("availability_zones", "list", None),
        "created_at": ("created_at", None, None),
        "description": ("description", None, None),
        "dns_domain": ("dns_domain", None, None),
        "id": ("id", None, None),
        "ipv4_address_scope_id": ("ipv4_address_scope", None, None),
        "ipv6_address_scope_id": ("ipv6_address_scope", None, None),
        "is_admin_state_up": ("admin_state_up", "bool", None),
        "is_default": ("is_default", "bool", None),
        "is_port_security_enabled": ("port_security_enabled", "bool", False),
        "is_router_external": ("router:external", "bool", False),
        "is_shared": ("shared", "bool", None),
        "is_vlan_qinq": ("vlan_qinq", "bool", None),
        "is_vlan_transparent": ("vlan_transparent", "bool", None),
        "mtu": ("mtu", "int", None),
        "name": ("name", None, None),
        "project_id": ("project_id", None, None),
        "provider_network_type": ("provider:network_type", None, None),
        "provider_physical_network": ("provider:physical_network", None, None),
        "provider_segmentation_id": ("provider:segmentation_id", None, None),
        "pvlan": ("pvlan", "bool", None),
        "qos_policy_id": ("qos_policy_id", None, None),
        "revision_number": ("revision_number", "int", None),
        "segments": ("segments", "list", None),
        "status": ("status", None, None),
        "subnet_ids": ("subnets", "list", None),
        "tags": ("tags", "list", []),
        "updated_at": ("updated_at", None, None),
    }
    actual = {}
    for name, (module, descriptor, _) in attrs.items():
        if not isinstance(descriptor, ast.Call):
            continue
        if source.resolve(module, descriptor.func) not in {
            "openstack.resource.Body", "openstack.fields.Body"
        }:
            continue
        if len(descriptor.args) != 1:
            raise ValueError("unsupported Network response field name")
        keywords = [keyword.arg for keyword in descriptor.keywords]
        if (len(keywords) != len(set(keywords))
                or set(keywords) - {"type", "default"}):
            raise ValueError("unsupported Network descriptor options")
        typed = next((keyword.value for keyword in descriptor.keywords
                      if keyword.arg == "type"), None)
        response_type = (source.resolve(module, typed).removeprefix("builtins.")
                         if typed is not None else None)
        default = next((keyword.value for keyword in descriptor.keywords
                        if keyword.arg == "default"), None)
        actual[name] = (source.literal(module, descriptor.args[0]), response_type,
                        source.literal(module, default) if default else None)
    if actual != expected:
        raise ValueError("unsupported Network declared response descriptors")
    init = source.anchor("openstack/fields.py", "_BaseComponent.__init__")
    args = init.args.posonlyargs + init.args.args
    defaults = dict(zip((arg.arg for arg in args[-len(init.args.defaults):]),
                        init.args.defaults))
    for name, expected_default in (("default", None), ("coerce_to_default", False),
                                   ("alternate_id", False), ("list_type", None)):
        if source.literal("openstack.fields", defaults[name]) is not expected_default:
            raise ValueError("unsupported Network implicit descriptor default")
    getter = source.anchor("openstack/fields.py", "_BaseComponent.__get__")
    null_returns = [node for node in getter.body if isinstance(node, ast.If)
                    and isinstance(node.test, ast.Compare)
                    and isinstance(node.test.left, ast.Name)
                    and node.test.left.id == "value"
                    and len(node.test.ops) == 1 and isinstance(node.test.ops[0], ast.Is)
                    and len(node.test.comparators) == 1
                    and isinstance(node.test.comparators[0], ast.Constant)
                    and node.test.comparators[0].value is None]
    if (len(null_returns) != 1 or len(null_returns[0].body) != 1
            or not isinstance(null_returns[0].body[0], ast.Return)
            or not isinstance(null_returns[0].body[0].value, ast.Constant)
            or null_returns[0].body[0].value.value is not None):
        raise ValueError("unsupported Network None response shortcut")
    conversion = source.anchor("openstack/fields.py", "_convert_type")
    bool_branches = [node for node in ast.walk(conversion) if isinstance(node, ast.If)
                     and isinstance(node.test, ast.Call)
                     and isinstance(node.test.func, ast.Name)
                     and node.test.func.id == "issubclass"
                     and len(node.test.args) == 2
                     and isinstance(node.test.args[0], ast.Name)
                     and node.test.args[0].id == "data_type"
                     and isinstance(node.test.args[1], ast.Name)
                     and node.test.args[1].id == "bool"]
    if len(bool_branches) != 1 or ast.dump(bool_branches[0].body[0]) != ast.dump(
        ast.Return(value=ast.Call(func=ast.Name(id="data_type", ctx=ast.Load()),
                                 args=[ast.Name(id="value", ctx=ast.Load())], keywords=[]))
    ) or len(bool_branches[0].body) != 1:
        raise ValueError("unsupported Network boolean conversion")
    expected_local = {name: {"field": field, "response_type": response_type}
                      for name, (field, response_type, _) in expected.items()
                      if name in {"availability_zone_hints", "availability_zones",
                                  "created_at", "dns_domain", "is_default",
                                  "is_vlan_qinq", "is_vlan_transparent", "mtu", "pvlan",
                                  "qos_policy_id", "revision_number", "segments",
                                  "subnet_ids", "updated_at"}}
    if body != expected_local:
        raise ValueError("unsupported Network local Body classification")


def router_body_descriptors(source, attrs, body, query):
    """Keep overridden response keys and deprecated tenant scope independent."""
    expected = {
        "availability_zone_hints": ("availability_zone_hints", "list", None, {}),
        "availability_zones": ("availability_zones", "list", None, {}),
        "created_at": ("created_at", None, None, {}),
        "description": ("description", None, None, {}),
        "enable_ndp_proxy": ("enable_ndp_proxy", "bool", None, {}),
        "evpn_vni": ("evpn_vni", "int", None, {}),
        "external_gateway_info": ("external_gateway_info", "dict", None, {}),
        "flavor_id": ("flavor_id", None, None, {}),
        "id": ("id", None, None, {}),
        "is_admin_state_up": ("admin_state_up", "bool", None, {}),
        "is_distributed": ("distributed", "bool", None, {}),
        "is_ha": ("ha", "bool", None, {}),
        "name": ("name", None, None, {}),
        "project_id": ("project_id", None, None, {"alias": "tenant_id"}),
        "revision_number": ("revision", "int", None, {}),
        "routes": ("routes", "list", None, {}),
        "status": ("status", None, None, {}),
        "tags": ("tags", "list", [], {}),
        "tenant_id": ("tenant_id", None, None, {"deprecated": True}),
        "updated_at": ("updated_at", None, None, {}),
    }
    actual = {}
    for name, (module, descriptor, _) in attrs.items():
        if not isinstance(descriptor, ast.Call):
            continue
        if source.resolve(module, descriptor.func) not in {
            "openstack.resource.Body", "openstack.fields.Body"
        }:
            continue
        if len(descriptor.args) != 1:
            raise ValueError("unsupported Router response field name")
        options = [keyword.arg for keyword in descriptor.keywords]
        if (len(options) != len(set(options))
                or set(options) - {"type", "default", "alias", "deprecated"}):
            raise ValueError("unsupported Router descriptor options")
        typed = next((keyword.value for keyword in descriptor.keywords
                      if keyword.arg == "type"), None)
        response_type = (source.resolve(module, typed).removeprefix("builtins.")
                         if typed is not None else None)
        default = next((keyword.value for keyword in descriptor.keywords
                        if keyword.arg == "default"), None)
        additional = {keyword.arg: source.literal(module, keyword.value)
                      for keyword in descriptor.keywords
                      if keyword.arg not in {"type", "default"}}
        actual[name] = (source.literal(module, descriptor.args[0]), response_type,
                        source.literal(module, default) if default else None,
                        additional)
    if actual != expected:
        raise ValueError("unsupported Router declared response descriptors")
    expected_query = {name: name for name in (
        "description", "fields", "flavor_id", "id", "name", "status",
        "project_id", "sort_key", "sort_dir", "limit", "marker", "tags"
    )}
    expected_query.update({
        "is_admin_state_up": "admin_state_up", "is_distributed": "distributed",
        "is_ha": "ha", "any_tags": "tags-any", "not_tags": "not-tags",
        "not_any_tags": "not-tags-any",
    })
    if query != expected_query:
        raise ValueError("unsupported Router query classification")
    expected_local = {
        name: {"field": field, "response_type": response_type}
        for name, (field, response_type, _, _) in expected.items()
        if name not in query
    }
    if body != expected_local:
        raise ValueError("unsupported Router local Body classification")
    if source.mro(ROUTER_RESOURCE) != [
        ROUTER_RESOURCE, "openstack.network.v2._base.NetworkResource",
        "openstack.resource.Resource", "builtins.dict",
        "openstack.network.v2._base.TagMixinNetwork",
        "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol",
        "typing.Protocol",
    ]:
        raise ValueError("unsupported Router inheritance")
    init = source.anchor("openstack/fields.py", "_BaseComponent.__init__")
    args = init.args.posonlyargs + init.args.args
    defaults = dict(zip((arg.arg for arg in args[-len(init.args.defaults):]),
                        init.args.defaults))
    for name, expected_default in (("default", None), ("coerce_to_default", False),
                                   ("alternate_id", False), ("list_type", None)):
        if source.literal("openstack.fields", defaults[name]) is not expected_default:
            raise ValueError("unsupported Router implicit descriptor default")
    getter = source.anchor("openstack/fields.py", "_BaseComponent.__get__")
    null_returns = [node for node in getter.body if isinstance(node, ast.If)
                    and isinstance(node.test, ast.Compare)
                    and isinstance(node.test.left, ast.Name)
                    and node.test.left.id == "value"
                    and len(node.test.ops) == 1 and isinstance(node.test.ops[0], ast.Is)
                    and len(node.test.comparators) == 1
                    and isinstance(node.test.comparators[0], ast.Constant)
                    and node.test.comparators[0].value is None]
    if (len(null_returns) != 1 or len(null_returns[0].body) != 1
            or not isinstance(null_returns[0].body[0], ast.Return)
            or not isinstance(null_returns[0].body[0].value, ast.Constant)
            or null_returns[0].body[0].value.value is not None):
        raise ValueError("unsupported Router None response shortcut")
    conversion = source.anchor("openstack/fields.py", "_convert_type")
    bool_branches = [node for node in ast.walk(conversion) if isinstance(node, ast.If)
                     and isinstance(node.test, ast.Call)
                     and isinstance(node.test.func, ast.Name)
                     and node.test.func.id == "issubclass"
                     and len(node.test.args) == 2
                     and isinstance(node.test.args[0], ast.Name)
                     and node.test.args[0].id == "data_type"
                     and isinstance(node.test.args[1], ast.Name)
                     and node.test.args[1].id == "bool"]
    if len(bool_branches) != 1 or ast.dump(bool_branches[0].body[0]) != ast.dump(
        ast.Return(value=ast.Call(func=ast.Name(id="data_type", ctx=ast.Load()),
                                 args=[ast.Name(id="value", ctx=ast.Load())], keywords=[]))
    ) or len(bool_branches[0].body) != 1:
        raise ValueError("unsupported Router boolean conversion")


def security_group_body_descriptors(source, attrs, body, query):
    """Keep queried tenant/revision fields outside the three raw predicates."""
    expected = {
        "created_at": ("created_at", None, None, {}),
        "description": ("description", None, None, {}),
        "id": ("id", None, None, {}),
        "is_shared": ("shared", "bool", None, {}),
        "name": ("name", None, None, {}),
        "project_id": ("project_id", None, None, {"alias": "tenant_id"}),
        "revision_number": ("revision_number", "int", None, {}),
        "security_group_rules": ("security_group_rules", "list", None, {}),
        "stateful": ("stateful", None, None, {}),
        "tags": ("tags", "list", [], {}),
        "tenant_id": ("tenant_id", None, None, {"deprecated": True}),
        "updated_at": ("updated_at", None, None, {}),
    }
    actual = {}
    for name, (module, descriptor, _) in attrs.items():
        if not isinstance(descriptor, ast.Call):
            continue
        if source.resolve(module, descriptor.func) not in {
            "openstack.resource.Body", "openstack.fields.Body"
        }:
            continue
        if len(descriptor.args) != 1:
            raise ValueError("unsupported SecurityGroup response field name")
        options = [keyword.arg for keyword in descriptor.keywords]
        if (len(options) != len(set(options))
                or set(options) - {"type", "default", "alias", "deprecated"}):
            raise ValueError("unsupported SecurityGroup descriptor options")
        typed = next((keyword.value for keyword in descriptor.keywords
                      if keyword.arg == "type"), None)
        response_type = (source.resolve(module, typed).removeprefix("builtins.")
                         if typed is not None else None)
        default = next((keyword.value for keyword in descriptor.keywords
                        if keyword.arg == "default"), None)
        additional = {keyword.arg: source.literal(module, keyword.value)
                      for keyword in descriptor.keywords
                      if keyword.arg not in {"type", "default"}}
        actual[name] = (source.literal(module, descriptor.args[0]), response_type,
                        source.literal(module, default) if default else None,
                        additional)
    if actual != expected:
        raise ValueError("unsupported SecurityGroup declared response descriptors")
    expected_query = {name: name for name in (
        "description", "fields", "id", "name", "stateful", "project_id",
        "tenant_id", "revision_number", "sort_dir", "sort_key", "limit",
        "marker", "tags"
    )}
    expected_query.update({
        "is_shared": "shared", "any_tags": "tags-any", "not_tags": "not-tags",
        "not_any_tags": "not-tags-any",
    })
    if query != expected_query:
        raise ValueError("unsupported SecurityGroup query classification")
    expected_local = {
        name: {"field": field, "response_type": response_type}
        for name, (field, response_type, _, _) in expected.items()
        if name not in query
    }
    if body != expected_local:
        raise ValueError("unsupported SecurityGroup local Body classification")
    if source.mro(SECURITY_GROUP_RESOURCE) != [
        SECURITY_GROUP_RESOURCE, "openstack.network.v2._base.NetworkResource",
        "openstack.resource.Resource", "builtins.dict",
        "openstack.network.v2._base.TagMixinNetwork",
        "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol",
        "typing.Protocol",
    ]:
        raise ValueError("unsupported SecurityGroup inheritance")
    init = source.anchor("openstack/fields.py", "_BaseComponent.__init__")
    args = init.args.posonlyargs + init.args.args
    defaults = dict(zip((arg.arg for arg in args[-len(init.args.defaults):]),
                        init.args.defaults))
    for name, expected_default in (("default", None), ("coerce_to_default", False),
                                   ("alternate_id", False), ("list_type", None)):
        if source.literal("openstack.fields", defaults[name]) is not expected_default:
            raise ValueError("unsupported SecurityGroup implicit descriptor default")
    getter = source.anchor("openstack/fields.py", "_BaseComponent.__get__")
    null_returns = [node for node in getter.body if isinstance(node, ast.If)
                    and isinstance(node.test, ast.Compare)
                    and isinstance(node.test.left, ast.Name)
                    and node.test.left.id == "value"
                    and len(node.test.ops) == 1 and isinstance(node.test.ops[0], ast.Is)
                    and len(node.test.comparators) == 1
                    and isinstance(node.test.comparators[0], ast.Constant)
                    and node.test.comparators[0].value is None]
    if (len(null_returns) != 1 or len(null_returns[0].body) != 1
            or not isinstance(null_returns[0].body[0], ast.Return)
            or not isinstance(null_returns[0].body[0].value, ast.Constant)
            or null_returns[0].body[0].value.value is not None):
        raise ValueError("unsupported SecurityGroup None response shortcut")


def trunk_body_descriptors(source, attrs, body, query):
    """Keep query-mapped subports distinct from raw id and deprecated tenant."""
    expected = {
        "id": ("id", None, None, {}),
        "name": ("name", None, None, {}),
        "project_id": ("project_id", None, None, {"alias": "tenant_id"}),
        "tenant_id": ("tenant_id", None, None, {"deprecated": True}),
        "description": ("description", None, None, {}),
        "is_admin_state_up": ("admin_state_up", "bool", None, {}),
        "port_id": ("port_id", None, None, {}),
        "status": ("status", None, None, {}),
        "sub_ports": ("sub_ports", "list", None, {}),
        "tags": ("tags", "list", [], {}),
    }
    actual = {}
    for name, (module, descriptor, _) in attrs.items():
        if not isinstance(descriptor, ast.Call):
            continue
        if source.resolve(module, descriptor.func) not in {
            "openstack.resource.Body", "openstack.fields.Body"
        }:
            continue
        if len(descriptor.args) != 1:
            raise ValueError("unsupported Trunk response field name")
        options = [keyword.arg for keyword in descriptor.keywords]
        if (len(options) != len(set(options))
                or set(options) - {"type", "default", "alias", "deprecated"}):
            raise ValueError("unsupported Trunk descriptor options")
        typed = next((keyword.value for keyword in descriptor.keywords
                      if keyword.arg == "type"), None)
        response_type = (source.resolve(module, typed).removeprefix("builtins.")
                         if typed is not None else None)
        default = next((keyword.value for keyword in descriptor.keywords
                        if keyword.arg == "default"), None)
        additional = {keyword.arg: source.literal(module, keyword.value)
                      for keyword in descriptor.keywords
                      if keyword.arg not in {"type", "default"}}
        actual[name] = (source.literal(module, descriptor.args[0]), response_type,
                        source.literal(module, default) if default else None,
                        additional)
    if actual != expected:
        raise ValueError("unsupported Trunk declared response descriptors")
    expected_query = {name: name for name in (
        "name", "description", "fields", "port_id", "status", "sub_ports",
        "project_id", "limit", "marker", "tags"
    )}
    expected_query.update({
        "is_admin_state_up": "admin_state_up", "any_tags": "tags-any",
        "not_tags": "not-tags", "not_any_tags": "not-tags-any",
    })
    if query != expected_query:
        raise ValueError("unsupported Trunk query classification")
    expected_local = {
        name: {"field": field, "response_type": response_type}
        for name, (field, response_type, _, _) in expected.items()
        if name not in query
    }
    if body != expected_local:
        raise ValueError("unsupported Trunk local Body classification")
    if source.mro(TRUNK_RESOURCE) != [
        TRUNK_RESOURCE, "openstack.resource.Resource", "builtins.dict",
        "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol",
        "typing.Protocol",
    ]:
        raise ValueError("unsupported Trunk inheritance")
    for name, expected_value in {
        "base_path": "/trunks", "resource_key": "trunk",
        "resources_key": "trunks", "allow_list": True,
        "_allow_unknown_attrs_in_body": True,
        "_store_unknown_attrs_as_properties": False, "_max_microversion": None,
    }.items():
        if source.literal(*attrs[name][:2]) != expected_value:
            raise ValueError("unsupported Trunk inherited resource policy")
    init = source.anchor("openstack/fields.py", "_BaseComponent.__init__")
    args = init.args.posonlyargs + init.args.args
    defaults = dict(zip((arg.arg for arg in args[-len(init.args.defaults):]),
                        init.args.defaults))
    for name, expected_default in (("default", None), ("coerce_to_default", False),
                                   ("alternate_id", False), ("list_type", None)):
        if source.literal("openstack.fields", defaults[name]) is not expected_default:
            raise ValueError("unsupported Trunk implicit descriptor default")
    getter = source.anchor("openstack/fields.py", "_BaseComponent.__get__")
    null_returns = [node for node in getter.body if isinstance(node, ast.If)
                    and isinstance(node.test, ast.Compare)
                    and isinstance(node.test.left, ast.Name)
                    and node.test.left.id == "value"
                    and len(node.test.ops) == 1 and isinstance(node.test.ops[0], ast.Is)
                    and len(node.test.comparators) == 1
                    and isinstance(node.test.comparators[0], ast.Constant)
                    and node.test.comparators[0].value is None]
    if (len(null_returns) != 1 or len(null_returns[0].body) != 1
            or not isinstance(null_returns[0].body[0], ast.Return)
            or not isinstance(null_returns[0].body[0].value, ast.Constant)
            or null_returns[0].body[0].value.value is not None):
        raise ValueError("unsupported Trunk None response shortcut")


def extract(root, target="subnet"):
    resource, files, anchors, sdk_package = TARGETS[target]
    source = Source(root, files)
    attrs = source.effective_attributes(resource)
    module, query_call, _ = attrs["_query_mapping"]
    if not isinstance(query_call, ast.Call) or source.resolve(module, query_call.func) != "openstack.resource.QueryParameters":
        raise ValueError("unsupported query mapping declaration")
    init = source.anchor("openstack/resource.py", "QueryParameters.__init__")
    pagination = dict(zip(
        (arg.arg for arg in init.args.kwonlyargs), init.args.kw_defaults
    ))["include_pagination_defaults"]
    include_defaults = source.literal("openstack.resource", pagination)
    keywords = {}
    for keyword in query_call.keywords:
        if keyword.arg is None:
            expansion = source.literal(module, keyword.value)
            if not isinstance(expansion, dict):
                raise ValueError("non-dictionary query expansion")
            keywords.update(expansion)
        else:
            keywords[keyword.arg] = source.literal(module, keyword.value)
    include_defaults = keywords.pop("include_pagination_defaults", include_defaults)
    if not isinstance(include_defaults, bool):
        raise ValueError("non-boolean pagination default")
    defaults = [node for node in ast.walk(init) if isinstance(node, ast.If)
                and isinstance(node.test, ast.Name) and node.test.id == "include_pagination_defaults"]
    if len(defaults) != 1 or len(defaults[0].body) != 1:
        raise ValueError("unsupported pagination defaults implementation")
    update = defaults[0].body[0]
    if (not isinstance(update, ast.Expr) or not isinstance(update.value, ast.Call)
            or len(update.value.args) != 1 or not isinstance(update.value.func, ast.Attribute)
            or update.value.func.attr != "update"):
        raise ValueError("unsupported pagination defaults mapping")
    mappings = source.literal("openstack.resource", update.value.args[0]) if include_defaults else {}
    for name in query_call.args:
        value = source.literal(module, name)
        if not isinstance(value, str):
            raise ValueError("non-string query key")
        mappings[value] = value
    mappings.update(keywords)
    query, formats = {}, {}
    for key, value in mappings.items():
        if isinstance(value, dict):
            if set(value) - {"name", "format"}:
                raise ValueError("unsupported query formatting policy")
            wire = value.get("name", key)
            if value.get("format") is not None:
                formats[key] = value["format"]
        else:
            wire = value
        if not isinstance(key, str) or not isinstance(wire, str):
            raise ValueError("non-string query mapping")
        query[key] = wire
    body, uri = {}, {}
    for name, (module, value, _) in attrs.items():
        if not isinstance(value, ast.Call):
            continue
        kind = source.resolve(module, value.func)
        if kind not in {"openstack.fields.Body", "openstack.fields.URI", "openstack.resource.Body", "openstack.resource.URI"}:
            continue
        if kind.endswith(".Body") and name in query:
            continue
        if len(value.args) != 1:
            raise ValueError("unsupported field name declaration")
        field = source.literal(module, value.args[0])
        typed = next((kw.value for kw in value.keywords if kw.arg == "type"), None)
        response_type = source.resolve(module, typed).removeprefix("builtins.") if typed else None
        (body if kind.endswith(".Body") else uri)[name] = {
            "field": field, "response_type": response_type
        }
    if target == "trunk":
        trunk_body_descriptors(source, attrs, body, query)
    elif target == "network":
        network_body_descriptors(source, attrs, body)
    elif target == "router":
        router_body_descriptors(source, attrs, body, query)
    elif target == "security_group":
        security_group_body_descriptors(source, attrs, body, query)
    elif target == "secret":
        keymanager_body_accessors(source, attrs, body, "Secret", "secret_id", "secret_ref")
    elif target == "container":
        keymanager_body_accessors(source, attrs, body, "Container", "container_id", "container_ref")
    elif target == "order":
        order_body_accessors(source, attrs, body)
    resource_controls = control_arguments(
        source.anchor("openstack/resource.py", "Resource.list"), {"cls"}
    )
    proxy_controls = control_arguments(
        source.anchor("openstack/proxy.py", "Proxy._list"), {"self"}
    )
    unknown_filters, query_collision = implementation_policies(source)
    proof = []
    for path, symbol in anchors:
        node = source.anchor(path, symbol)
        proof.append({
            "source": path, "symbol": symbol,
            "line": node.lineno, "end_line": node.end_lineno,
            "ast_sha256": hashlib.sha256(ast.dump(
                node, annotate_fields=True, include_attributes=False
            ).encode("utf-8")).hexdigest(),
        })
    return {
        "schema_version": 1, "source_pin": PIN, "resource": resource,
        "sdk_package": sdk_package,
        "base_path": source.literal(*attrs["base_path"][:2]),
        "envelope": source.literal(*attrs["resources_key"][:2]),
        "class_bases": [ast.unparse(base) for base in source.classes[resource].bases],
        "mro": source.mro(resource),
        "query": query, "query_formats": formats, "body": body, "uri": uri,
        "unknown_filters": unknown_filters,
        "query_collision": query_collision,
        "counts": {
            "canonical_query": len(query),
            "accepted_query": len(set(query) | set(query.values())),
            "local_body": len(body),
        },
        "source_controls": {"resource_list": resource_controls, "proxy_list": proxy_controls},
        "reserved": sorted(set(resource_controls) | set(proxy_controls)),
        "proof": {
            "ast_algorithm": 'sha256(ast.dump(node, annotate_fields=True, include_attributes=False).encode("utf-8"))',
            "python_parser": ".".join(map(str, sys.version_info[:2])),
            "nodes": proof,
            "files": {path: hashlib.sha256(raw).hexdigest() for path, raw in source.raw.items()},
        },
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path)
    parser.add_argument("--target", choices=tuple(TARGETS))
    parser.add_argument("--resource", choices=tuple(spec[0] for spec in TARGETS.values()))
    args = parser.parse_args()
    target = args.target
    if args.resource:
        resource_target = next(name for name, spec in TARGETS.items()
                               if spec[0] == args.resource)
        if target is not None and target != resource_target:
            parser.error("--resource and --target select different resources")
        target = resource_target
    try:
        data = json.dumps(extract(args.source, target or "subnet"), indent=2, sort_keys=True) + "\n"
    except (OSError, SyntaxError, ValueError, KeyError, TypeError, AttributeError) as error:
        parser.exit(1, "Python filter source: " + str(error) + "\n")
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(data, encoding="utf-8")
    else:
        sys.stdout.write(data)


if __name__ == "__main__":
    main()
