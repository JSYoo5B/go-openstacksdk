# Native

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/servers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkMode`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworks`
- `github.com/JSYoo5B/go-openstacksdk.WithDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithoutDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateHintOpts`

- Ordered NICs preserve network UUID and port combinations, literal fixed IP, fixed-IP-only/tag-only/empty objects, and omitted versus explicit empty or null tag values in the HTTP body. IDs bypass Neutron lookup and need no Network endpoint.
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Mode auto/none requires selected Compute 2.37+, and every present tag, including empty or null, requires selected 2.42+. Invalid options/Refs/version gates fail before dependency lookup; the shared client version and Nova headers are preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connectiontest/connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Create POSTs {"server": ...} with required name, base64 user_data, named security groups and auto/none or structured networks, places extension fields inside the single server envelope and scheduler hints at top-level os:scheduler_hints, rejects invalid networks, invalid hints, collisions and nil options before HTTP, and accepts 200 and 202.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.
- The remaining CreateOpts fields keep their pinned JSON names and omitempty rules: availability zone, metadata, base64 personality files, admin password, access IPs, tags, hostname, block_device_mapping_v2 with boot_index/delete_on_termination always sent, OS-DCF:diskConfig, hypervisor_hostname and structured or none networks; the selected client microversion header reaches the request.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCreateFullOptionsAndMicroversion`.

Python/Go differences:

- The SDK owns concrete NIC inputs and an Optional tag field; request.Present/Null distinguish omitted, empty and null without a caller builder. The owned network body overlay avoids native empty-string tag omission.
- Network and port are explicit safe Ref IDs/names. Connection adds port name resolution, while Python cloud port/port-id values are literal IDs. The SDK copies inputs instead of mutating Python NIC dictionaries.
- The final selection option wins. Empty network/NIC selections are invalid options rather than Python empty-list fallback.
- Connection supplies configured defaults without an application builder/resolver. Explicit Ref IDs bypass Neutron and Ref names use exact-name lookup. Configured YAML/role defaults match name OR ID through shared successful role discovery, while explicit typed default Name remains fresh. Discovery errors preserve their causes and are not cached as empty success.
- Version gates follow supports_microversion/require_microversion with the client's selected microversion as the adapter default. Python sends an unselected request at min(Server._max_microversion 2.100, advertised maximum); Go keeps the base version and raises only the create POST to the gate it needs (2.37 for auto, 2.42 for tags), so other body fields keep their base-version validation. A selection below the gate decides without discovery. Python's discovery failure yields unsupported; Go treats only a clean 404/405 or missing bound as unsupported and returns other discovery failures before the POST rather than silently changing the networks body. Explicit WithNetworkMode keeps its selected-2.37 check without discovery, which Python's cloud layer does not expose.

### `gophercloud:compute/v2/servers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.RawClient`

- Get sends GET servers/{raw id} with service MoreHeaders and live token, accepts 200 and 203, decodes Server with an empty image string as nil Image, and keeps other statuses as native errors under Get/servers context.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListQuery`

- List GETs servers/detail with the ListOpts query including all_tenants and extension values, follows the servers_links next href verbatim and sends no further page after the caller stops.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerListsQueryPagingAndStop`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ListSimple`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListSimple`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListSimpleOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListSimpleQuery`

- ListSimple GETs servers with the same ListOpts query and follows the servers_links next href verbatim.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerListsQueryPagingAndStop`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUpdateOptions`

- Update PUTs {"server": ...} with omitted empty name/access IPs and an optional hostname, returns the decoded server and accepts only 200.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Delete`

- Delete sends DELETE servers/{raw id} and accepts the native default 202 and 204.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadata`

- Metadata GETs servers/{id}/metadata, returns the decoded metadata map and accepts only 200.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Metadatum`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadatum`

- Metadatum GETs servers/{id}/metadata/{raw key}, returns the decoded meta map and accepts only 200.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ResetMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ResetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithResetMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithResetMetadataField`

- ResetMetadata PUTs {"metadata": opts} replacing every item, returns the server metadata map, rejects a metadata key collision and accepts only 200.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.UpdateMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.UpdateMetadata`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUpdateMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUpdateMetadataField`

- UpdateMetadata POSTs {"metadata": opts} merging items, adds extension fields beside metadata because the typed map is not a generic envelope, rejects nil options and accepts only 200.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.CreateMetadatum`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.CreateMetadatum`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateMetadatumOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateMetadatumField`

- CreateMetadatum requires exactly one pair before HTTP, PUTs {"meta": opts} to servers/{id}/metadata/{key}, returns the server meta map and accepts only 200.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.DeleteMetadatum`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.DeleteMetadatum`

- DeleteMetadatum sends DELETE servers/{id}/metadata/{raw key} and accepts the native default 202 and 204.
  Tests: `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataRoutesBodiesAndDecode`, `compute/v2/servers/native_metadata_contracts_test.go:TestNativeServerMetadataStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server metadata call only; Python compute server metadata parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ChangeAdminPassword`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ChangeAdminPassword`

- ChangeAdminPassword POSTs {"changePassword":{"adminPass":...}} to servers/{id}/action and accepts only the POST default 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Reboot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Reboot`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRebootOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRebootField`

- Reboot POSTs {"reboot":{"type":...}} with extensions inside the action object, rejects a missing type, a type extension collision before HTTP and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Rebuild`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Rebuild`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRebuildOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRebuildField`

- Rebuild POSTs {"rebuild":{...}} always including imageRef, validates DiskConfig before HTTP, decodes the response server and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Resize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Resize`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithResizeOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithResizeField`

- Resize POSTs {"resize":{"flavorRef":...}} with optional OS-DCF:diskConfig, rejects a missing flavor or invalid DiskConfig before HTTP and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ConfirmResize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConfirmResize`

- ConfirmResize POSTs {"confirmResize":null} and accepts only 201/202/204.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.RevertResize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.RevertResize`

- RevertResize POSTs {"revertResize":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.CreateImage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.CreateImage`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateImageOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateImageField`

- CreateImage POSTs {"createImage":{...}}, requires a name before HTTP, accepts only 202 and extracts the image ID from Location before microversion 2.45 or from a JSON body image_id from 2.45, returning an empty ID for a non-JSON body and an error after 202 for a missing or bad version header or Location.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerCreateImageIDByMicroversion`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Rescue`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Rescue`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRescueOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRescueField`

- Rescue POSTs {"rescue":{...}}, returns the response adminPass, rejects nil options before HTTP and accepts only 200.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Unrescue`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unrescue`

- Unrescue POSTs {"unrescue":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Start`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Start`

- Start POSTs {"os-start":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Stop`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Stop`

- Stop POSTs {"os-stop":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Pause`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Pause`

- Pause POSTs {"pause":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Unpause`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unpause`

- Unpause POSTs {"unpause":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Suspend`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Suspend`

- Suspend POSTs {"suspend":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Resume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Resume`

- Resume POSTs {"resume":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Lock`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Lock`

- Lock POSTs {"lock":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Unlock`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unlock`

- Unlock POSTs {"unlock":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Shelve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Shelve`

- Shelve POSTs {"shelve":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ShelveOffload`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ShelveOffload`

- ShelveOffload POSTs {"shelveOffload":null} and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Unshelve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unshelve`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUnshelveOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUnshelveField`

- Unshelve POSTs {"unshelve":null} without an availability zone (extensions beside it) or {"unshelve":{"availability_zone":...}} (extensions inside), rejects an availability_zone extension collision before HTTP and accepts only 201/202.
  Tests: `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsStrictStatuses`, `compute/v2/servers/native_action_contracts_test.go:TestNativeServerActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server user action call only; Python compute server action parity and admin actions are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ListAddresses`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListAddresses`

- ListAddresses GETs servers/{id}/ips once, yields the single addresses map, yields nothing for an empty map, surfaces io.EOF for a bodyless 204 and returns native pager status errors (200/204/300) without operation context.
  Tests: `compute/v2/servers/native_address_console_contracts_test.go:TestNativeServerAddressesSinglePageStreams`.

Python/Go differences:

- This declaration maps the Gophercloud native server address, console output and status wait calls only; Python wait_for_server failure states and timeouts are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, polling and decoding are unchanged.

### `gophercloud:compute/v2/servers.ListAddressesByNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListAddressesByNetwork`

- ListAddressesByNetwork GETs servers/{id}/ips/{raw network} once, yields the rows of the single top-level network key, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors without operation context.
  Tests: `compute/v2/servers/native_address_console_contracts_test.go:TestNativeServerAddressesSinglePageStreams`.

Python/Go differences:

- This declaration maps the Gophercloud native server address, console output and status wait calls only; Python wait_for_server failure states and timeouts are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, polling and decoding are unchanged.

### `gophercloud:compute/v2/servers.ShowConsoleOutput`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ShowConsoleOutput`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithShowConsoleOutputOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithShowConsoleOutputField`

- ShowConsoleOutput POSTs {"os-getConsoleOutput":{...}} omitting a zero length, places extensions inside the action object, rejects a length collision or nil option before HTTP, returns the output string and accepts only 200.
  Tests: `compute/v2/servers/native_address_console_contracts_test.go:TestNativeServerConsoleOutput`.

Python/Go differences:

- This declaration maps the Gophercloud native server address, console output and status wait calls only; Python wait_for_server failure states and timeouts are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, polling and decoding are unchanged.

### `gophercloud:compute/v2/servers.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForStatus`

- WaitForStatus GETs servers/{id} immediately and then once per second until the status matches exactly, treats ERROR as non-terminal, returns the context error on expiry and stops on the first GET error, all with operation context.
  Tests: `compute/v2/servers/native_address_console_contracts_test.go:TestNativeServerWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native server address, console output and status wait calls only; Python wait_for_server failure states and timeouts are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, polling and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servers/requests.go)
- [Create With Floating Ip](../../create-with-floating-ip.md)
- [Service Waits](../../../docs/service-waits.md)
