# Compute

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
for server in conn.compute.servers(status="ACTIVE"):
    print(server.id, server.name)
```

See `Example_computeFragment1` in [example_compute_fragment1_test.go](example_compute_fragment1_test.go), compiled by `go test`.

See `ExampleServers_Create` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

```python
server = conn.create_server(
    name="volume-server",
    flavor="small",
    boot_volume="root-volume",
    terminate_volume=False,
    network="private",
    auto_ip=False,
    wait=True,
)
```

See `ExampleWithBootVolume` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

```python
server = conn.create_server(
    name="new-volume-server",
    image="ubuntu",
    flavor="small",
    boot_from_volume=True,
    volume_size=50,
    terminate_volume=False,
    auto_ip=False,
    wait=True,
)
```

See `ExampleWithBootVolumeSize` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

```python
server = conn.create_server(
    name="grouped",
    image="ubuntu",
    flavor="small",
    group="anti-affinity",
    scheduler_hints={"different_host": ["other-id"]},
    volumes=["data"],
    userdata="#cloud-config\n",
    meta={"role": "web"},
)
```

See `ExampleWithServerGroup` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

Create property rules:

- Scheduler hints are sent in the top-level `OS-SCH-HNT:scheduler_hints`
  object, as Python's Server resource does. `WithServerGroup` always sets the
  `group` hint, replacing a `group` key from `WithSchedulerHints`. Group names
  are resolved by listing `os-server-groups` before the POST; IDs skip lookup.
- `WithDataVolumes` adds `block_device_mapping_v2` entries with
  `boot_index` -1 and `delete_on_termination` false. When booting from an
  image, a local image mapping with `boot_index` 0 and
  `delete_on_termination` true is added first. Python sends some boot indexes
  as strings; Go sends integers.
- `WithUserData` always base64-encodes the supplied bytes, even when they
  already decode as base64. `WithServerCount` sends `min_count`/`max_count`;
  without it both keys are omitted, whereas Python's cloud layer sends 1/1.
- `WithDescription` (2.19), `WithTags` (2.52), `WithHypervisorHostname` (2.74)
  and `WithHostname` (2.90) are checked against the client microversion before
  any request. `WithPersonality` is rejected locally at 2.57 and later.
- `WithBlockDeviceMappings` sends explicit `block_device_mapping_v2` entries
  with every per-mapping field (`BootIndex` may be `request.Null[int]()`);
  generated boot and data-volume mappings follow them. Tag needs 2.42 and
  volume type 2.67. `WithLegacyBlockDeviceMappings` sends the legacy
  `block_device_mapping` list and cannot be combined with any v2 mapping.
  `WithTrustedImageCertificates` needs 2.63.
- `Servers.CreateReservation` sends `return_reservation_id` and returns only
  the reservation ID; combine it with `WithServerCount` for several servers.
  `Servers.Create` with a count returns the single server Nova reports.
- `config_drive` is a boolean; volume-ID values are not accepted by Nova.
- Falsy values are not silently dropped: `WithConfigDrive(false)` sends
  `false`, and empty admin passwords, tags and hint keys are rejected.

Readiness errors and cleanup:

- `WithCreateRefresh` matches Python's `wait=False` path: after the POST it
  performs one GET and returns the refreshed server, or `ServerFaultError`
  when it is in ERROR. A failed GET returns the POST response with the error;
  the POST is never repeated. It cannot be combined with `WithWait`.
- `Service.CreateRefreshedServer` (and `Connection.CreateRefreshedServer`)
  adds Python's `add_server_interfaces` step: it returns the POST response,
  the refreshed server with the creation `adminPass` copied onto it, and the
  `ExpandServerInterfaces` address view. `CreateWithAutomaticFloatingIP` also
  copies the creation `adminPass` onto its returned `Server`.
- Only Python's falsy address maps (null or `{}`) are deleted under
  `DeleteServerWithoutAddresses`. A map with only empty rows sets `EmptyRows`
  and is kept; Python would instead try to add IPs to it.

- A server in ERROR returns `*compute.ServerFaultError`, which unwraps to
  `resource.FailedStateError`. Its message follows Python's two forms
  ("Compute service reports fault: ..." or "no further information
  available"), based on whether the raw `fault` object has a `message` key.
  `Server` is the observed server, like Python's `extra_data`.
- An ACTIVE server with no address rows returns
  `ServerAddressesUnavailableError`. By default no DELETE is sent. Selecting
  `compute.DeleteServerWithoutAddresses` (`WithServerWithoutAddressesPolicy`
  or `AutomaticServerCreateOptions.ServerWithoutAddresses`) sends exactly one
  DELETE, records the result in `Deletion`, and waits for 404 when the
  workflow waits. A failed DELETE is joined to the error and never retried.

See `ExampleServers_Wait` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

## References

- [Console Auth Token](console-auth-token.md)
- [Console Selection](console-selection.md)
- [..](../README.md)
- [User Read APIs](user-read-apis.md)
- [User Actions](user-actions.md)
- [Keypairs List Find](keypairs-list-find.md)
- [Keypairs Cloud](keypairs-cloud.md)
- [Keypairs Console](keypairs-console.md)
- [Flavor Records](flavor-records.md)
- [Flavor Cloud](flavor-cloud.md)
- [Availability Zones](availability-zones.md)
- [Flavor Extra Specs](flavor-extra-specs.md)
- [Cloud keypair](keypairs-cloud.md)
- [AddIPsToServer](server-ip-helpers.md)
- [AddIPList](server-ip-helpers.md)
- [Server Addresses](server-addresses.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
- [GetActiveServer](server-ready.md)
- [Service·Connection WaitForServer](server-ready.md)
- [pool·IP dispatch](server-ip-dispatch.md)
- [Compute](https://docs.openstack.org/openstacksdk/latest/user/proxies/compute.html)
- [Finding Identities](../docs/finding-identities.md)
- [Flavor Property And Native](flavor-property-and-native.md)
- [Server Default Network](server-default-network.md)
- [Server Network Interfaces](server-network-interfaces.md)
- [V2](v2/README.md)
- [compute_test.go](compute_test.go)
- [boot_volume_test.go](boot_volume_test.go)
- [new_boot_volume_test.go](new_boot_volume_test.go)
- [server_create_test.go](../connectiontest/server_create_test.go)
- [collections_test.go](../connectiontest/collections_test.go)
- [Service Waits](../docs/service-waits.md)
- [Server Ready](server-ready.md)
- [CreateWithFloatingIP](create-with-floating-ip.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Network Roles](../network/network-roles.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [Server Automatic Ip](server-automatic-ip.md)
- [Legacy Nova floating IP](server-nova-floating-ip.md)
- [Floating Ip Available](floating-ip-available.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Floating Ip Delete](floating-ip-delete.md)
- [Floating Ip Create](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
