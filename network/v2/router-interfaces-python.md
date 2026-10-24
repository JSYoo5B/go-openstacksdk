# Router Interfaces Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/add_gateway_to_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddGatewayRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithSourceLegacyGatewayRoutes`

- Default single-gateway intent uses stock PUT /routers/{id} external_gateway_info: addition nests preserved attrs and requires a nonblank string network_id to prevent accidental clearing; removal sends explicit null and rejects attributes rather than silently discarding them. The opt-in Source legacy routes send envelope-free kwargs without defaults; reserved proxy arguments are rejected.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterGatewayStockBridgeAndExplicitLegacyBodies`.
- Complete parsed response.json() values including objects/envelopes, lists, strings, bool, numbers and null are owned independently of actual Envelope/Header/status. No Router canonical projection or cache seed is applied. Empty/malformed accepted JSON carries actual proof and never fabricates an acknowledgement.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesWholeJSONValuesStrictStatusesAndDecodeEvidence`.
- Accepted/rejected physical Read/Close/cancel/source-changed-and-restored-on-Close errors preserve causes and actual body/header/status, remain terminal and never resend or become a clean missing result.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesPhysicalReadCloseCancelAndSourceFaultsRemainTerminal`.
- Selected source is captured before option callbacks; raw reference/field/header inputs and retained callback values are cloned. Protected header/body/option/source/context errors fail before HTTP, and ordinary API options use concrete library-owned profiles.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOptionsOwnSnapshotsAndContextSourcePreflight`.
- Native retry and reauthentication retain exact route/JSON, detect attempted body/source mutation, preserve the original native status cause and support independent concurrent results from shared owned options.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryReauthAndConcurrentInputsKeepOwnership`.
- Native retry cannot alter or omit selected network version/JSON content type or inject a version when originally absent, foreign-service version, authentication/framing headers. One actual503 remains terminal with original body/header/status and callback causes; ordinary retry header changes remain allowed.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryCannotChangeProtocolOrJSONHeaders`, `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOrdinaryRetryHeadersRemainNative`.

Python/Go differences:

- Go uses explicit literal router IDs with one-time URL escaping, complete UTF-8 JSON values and owned results rather than arbitrary Python object models. A RawResource port/subnet reference models Source Resource._get_id; ordinary Go JSON maps stay ordinary values rather than being treated as Resource instances. Whole Connection/Resource/session parity remains separately tracked.
- Pinned Source add/remove_gateway_router action paths are absent from Neutron27.0.0 stock L3, whose constrained neutron-lib3.21.1 action map registers interface actions only. Go deliberately bridges the named single-gateway intent to stock external_gateway_info updates. WithSourceLegacyGatewayRoutes(true) restores only the exact Source action path and kwargs for a supporting backend; this is not stock legacy-route support.
- Pinned Source gateway methods return response.json() without raise_from_response and Proxy.request defaults raise_exc=False, allowing final4xx/5xx JSON to masquerade as returned data. Go deliberately repairs this defect by retaining strict native HTTP errors in both default and legacy-route modes; HTTP error JSON is never declared SDK success.
- The stock bridge AddGatewayRecord requires network_id because an empty external_gateway_info object may clear a live gateway. Stock RemoveGatewayRecord has no attribute selection and rejects supplied fields instead of ignoring them. Source removal kwargs remain available only under explicit legacy route compatibility.
- Stock external_gateway_info/network_id update permits admin or the owning-project member. enable_snat/external_fixed_ips are admin-only attribute branches. Legacy paths have no corresponding stock27 policy; their backend authorization/support cannot be inferred from the stock bridge.

### `python:network/v2/add_interface_to_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePortID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnetID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceAdvertiseHost`

- One exact Source interface PUT performs no prefetch/envelope synthesis. Truthy port wins subnet; defaults/falsy port fall back to subnet, including null. Cached Resource truthiness remains separate from its nullable ID. Deprecated port_id/subnet_id aliases override current names regardless of option order. Add emits advertise_host only when truthy; removal rejects that unsupported keyword.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesSelectionDefaultsRenamedAliasesAndAdvertiseHost`.
- Complete parsed response.json() values including objects/envelopes, lists, strings, bool, numbers and null are owned independently of actual Envelope/Header/status. No Router canonical projection or cache seed is applied. Empty/malformed accepted JSON carries actual proof and never fabricates an acknowledgement.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesWholeJSONValuesStrictStatusesAndDecodeEvidence`.
- Accepted/rejected physical Read/Close/cancel/source-changed-and-restored-on-Close errors preserve causes and actual body/header/status, remain terminal and never resend or become a clean missing result.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesPhysicalReadCloseCancelAndSourceFaultsRemainTerminal`.
- Selected source is captured before option callbacks; raw reference/field/header inputs and retained callback values are cloned. Protected header/body/option/source/context errors fail before HTTP, and ordinary API options use concrete library-owned profiles.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOptionsOwnSnapshotsAndContextSourcePreflight`.
- Native retry and reauthentication retain exact route/JSON, detect attempted body/source mutation, preserve the original native status cause and support independent concurrent results from shared owned options.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryReauthAndConcurrentInputsKeepOwnership`.
- Native retry cannot alter or omit selected network version/JSON content type or inject a version when originally absent, foreign-service version, authentication/framing headers. One actual503 remains terminal with original body/header/status and callback causes; ordinary retry header changes remain allowed.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryCannotChangeProtocolOrJSONHeaders`, `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOrdinaryRetryHeadersRemainNative`.

Python/Go differences:

- Go uses explicit literal router IDs with one-time URL escaping, complete UTF-8 JSON values and owned results rather than arbitrary Python object models. A RawResource port/subnet reference models Source Resource._get_id; ordinary Go JSON maps stay ordinary values rather than being treated as Resource instances. Whole Connection/Resource/session parity remains separately tracked.
- Go preserves Source old-name argument precedence but emits no Python runtime deprecation warning. Inputs are JSON-domain references rather than Python custom objects, and invalid selected JSON values fail before HTTP; unused subnet references do not influence a truthy-port request.
- Both interface actions permit admin or the owning-project member under fixed Neutron27.0.0 stock policy. Advertise_host is a pinned Source input projection; Neutron27 base l3_db handler does not read that flag, so successful submission is not proof of EVPN host-route advertisement on a deployed backend.

### `python:network/v2/remove_gateway_from_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveGatewayRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterGatewayHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithSourceLegacyGatewayRoutes`

- Default single-gateway intent uses stock PUT /routers/{id} external_gateway_info: addition nests preserved attrs and requires a nonblank string network_id to prevent accidental clearing; removal sends explicit null and rejects attributes rather than silently discarding them. The opt-in Source legacy routes send envelope-free kwargs without defaults; reserved proxy arguments are rejected.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterGatewayStockBridgeAndExplicitLegacyBodies`.
- Complete parsed response.json() values including objects/envelopes, lists, strings, bool, numbers and null are owned independently of actual Envelope/Header/status. No Router canonical projection or cache seed is applied. Empty/malformed accepted JSON carries actual proof and never fabricates an acknowledgement.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesWholeJSONValuesStrictStatusesAndDecodeEvidence`.
- Accepted/rejected physical Read/Close/cancel/source-changed-and-restored-on-Close errors preserve causes and actual body/header/status, remain terminal and never resend or become a clean missing result.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesPhysicalReadCloseCancelAndSourceFaultsRemainTerminal`.
- Selected source is captured before option callbacks; raw reference/field/header inputs and retained callback values are cloned. Protected header/body/option/source/context errors fail before HTTP, and ordinary API options use concrete library-owned profiles.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOptionsOwnSnapshotsAndContextSourcePreflight`.
- Native retry and reauthentication retain exact route/JSON, detect attempted body/source mutation, preserve the original native status cause and support independent concurrent results from shared owned options.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryReauthAndConcurrentInputsKeepOwnership`.
- Native retry cannot alter or omit selected network version/JSON content type or inject a version when originally absent, foreign-service version, authentication/framing headers. One actual503 remains terminal with original body/header/status and callback causes; ordinary retry header changes remain allowed.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryCannotChangeProtocolOrJSONHeaders`, `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOrdinaryRetryHeadersRemainNative`.

Python/Go differences:

- Go uses explicit literal router IDs with one-time URL escaping, complete UTF-8 JSON values and owned results rather than arbitrary Python object models. A RawResource port/subnet reference models Source Resource._get_id; ordinary Go JSON maps stay ordinary values rather than being treated as Resource instances. Whole Connection/Resource/session parity remains separately tracked.
- Pinned Source add/remove_gateway_router action paths are absent from Neutron27.0.0 stock L3, whose constrained neutron-lib3.21.1 action map registers interface actions only. Go deliberately bridges the named single-gateway intent to stock external_gateway_info updates. WithSourceLegacyGatewayRoutes(true) restores only the exact Source action path and kwargs for a supporting backend; this is not stock legacy-route support.
- Pinned Source gateway methods return response.json() without raise_from_response and Proxy.request defaults raise_exc=False, allowing final4xx/5xx JSON to masquerade as returned data. Go deliberately repairs this defect by retaining strict native HTTP errors in both default and legacy-route modes; HTTP error JSON is never declared SDK success.
- The stock bridge AddGatewayRecord requires network_id because an empty external_gateway_info object may clear a live gateway. Stock RemoveGatewayRecord has no attribute selection and rejects supplied fields instead of ignoring them. Source removal kwargs remain available only under explicit legacy route compatibility.
- Stock external_gateway_info/network_id update permits admin or the owning-project member. enable_snat/external_fixed_ips are admin-only attribute branches. Legacy paths have no corresponding stock27 policy; their backend authorization/support cannot be inferred from the stock bridge.

### `python:network/v2/remove_interface_from_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePortID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnetID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceHeaders`

- One exact Source interface PUT performs no prefetch/envelope synthesis. Truthy port wins subnet; defaults/falsy port fall back to subnet, including null. Cached Resource truthiness remains separate from its nullable ID. Deprecated port_id/subnet_id aliases override current names regardless of option order. Add emits advertise_host only when truthy; removal rejects that unsupported keyword.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesSelectionDefaultsRenamedAliasesAndAdvertiseHost`.
- Complete parsed response.json() values including objects/envelopes, lists, strings, bool, numbers and null are owned independently of actual Envelope/Header/status. No Router canonical projection or cache seed is applied. Empty/malformed accepted JSON carries actual proof and never fabricates an acknowledgement.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesWholeJSONValuesStrictStatusesAndDecodeEvidence`.
- Accepted/rejected physical Read/Close/cancel/source-changed-and-restored-on-Close errors preserve causes and actual body/header/status, remain terminal and never resend or become a clean missing result.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesPhysicalReadCloseCancelAndSourceFaultsRemainTerminal`.
- Selected source is captured before option callbacks; raw reference/field/header inputs and retained callback values are cloned. Protected header/body/option/source/context errors fail before HTTP, and ordinary API options use concrete library-owned profiles.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOptionsOwnSnapshotsAndContextSourcePreflight`.
- Native retry and reauthentication retain exact route/JSON, detect attempted body/source mutation, preserve the original native status cause and support independent concurrent results from shared owned options.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryReauthAndConcurrentInputsKeepOwnership`.
- Native retry cannot alter or omit selected network version/JSON content type or inject a version when originally absent, foreign-service version, authentication/framing headers. One actual503 remains terminal with original body/header/status and callback causes; ordinary retry header changes remain allowed.
  Tests: `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesRetryCannotChangeProtocolOrJSONHeaders`, `network/v2/extensions/layer3/routers/python_interfaces_contracts_test.go:TestPythonRouterInterfacesOrdinaryRetryHeadersRemainNative`.

Python/Go differences:

- Go uses explicit literal router IDs with one-time URL escaping, complete UTF-8 JSON values and owned results rather than arbitrary Python object models. A RawResource port/subnet reference models Source Resource._get_id; ordinary Go JSON maps stay ordinary values rather than being treated as Resource instances. Whole Connection/Resource/session parity remains separately tracked.
- Go preserves Source old-name argument precedence but emits no Python runtime deprecation warning. Inputs are JSON-domain references rather than Python custom objects, and invalid selected JSON values fail before HTTP; unused subnet references do not influence a truthy-port request.
- Both interface actions permit admin or the owning-project member under fixed Neutron27.0.0 stock policy. Advertise_host is a pinned Source input projection; Neutron27 base l3_db handler does not read that flag, so successful submission is not proof of EVPN host-route advertisement on a deployed backend.

### `python:cloud/_network/NetworkCloudMixin/remove_router_interface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.RemoveRouterInterface`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveSubnetID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemovePortID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveCloudInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceOpts`

- RemoveInterfaceRecord sends PUT /v2.0/routers/{id}/remove_router_interface with subnet_id, or port_id when a port is given, and with neither it still sends subnet_id null.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBRemoveRouterInterfaceBodies`.
- Concrete SDK options own absent/null/raw JSON and cached SDK Resource references. Both subnet_id and port_id falsey rejects before network service/router ID validation or HTTP. Python-loaded float-underflow/empty values are falsey; nonnil Resources remain truthy even with null/false/zero/empty IDs. Plain JSON dictionaries remain body values rather than being treated as Resources.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveBothFalseyFirstAndHTTPZero`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`.
- Source performs no get_router/find/name resolution: strings are literal router IDs. A truthy port wins and causes only port_id to be sent, otherwise subnet_id. SDK-owned raw/PortRecord/SubnetRecord/RawResource conversion avoids application builders/per-record adapters and owns copied options, selected service/API/client/header state and immediately stopping callbacks after detected source changes.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveOwnsOptionRefsHeadersAndEscapedState`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCancellationNilOptionsAndStickySourceStopCallbacks`.
- Router.remove_interface evaluates actual response.json before cloud discards it and returns None. Successful Value is JSON null for arbitrary valid object/list/string/number/bool/null, while ResponseJSON/optional actual Wire/whole Envelope/header/status retain independent physical observations. Accepted empty204 or malformed JSON remains decode error; no synthetic router Resource/default or convergence claim is created.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveSourceNoneAfterArbitraryJSONAndActualCopies`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`.
- A closed new helper reuses existing interface reference/body/header/JSON and shared REST contracts, opting into Source100..399 and StopFaultRetryCallbacks. Existing RemoveInterfaceRecord default-subnet-null/200..399 and native status/required behavior remain unchanged; 199 lane isolation is verified without promoting another Source API.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveClosedSourceStatusAndLegacyIsolation`.
- Clean403/404/409 remains actual native error, not absence/success. Accepted and rejected physical read/Close/cancel/source faults preserve actual status/body/header and original causes with exactly one physical PUT/body Close and zero retry/backoff/reauth callbacks. Native401 ErrOriginal is explicitly inspected despite absent Unwrap, and ErrReauth preserves read/Close causes. Clean401 renewal and503 retry keep original request body/headers and actual final receipt.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePhysicalFaultsKeepActualProofAndPreventReplay`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCleanRetryAndTokenRenewalKeepCapturedBody`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemovePhysicalOptInNeverCallsHooks`.
- Only RemoveCloudInterfaceRecord normalizes explicit ISO8859-1/latin1, UTF16/32 with BOM or explicit LE/BE through the fixed shared responsejson.Normalize ABI. Parsed Value and optional object Wire.Body use normalized JSON; Envelope/Header/status/ResponseError.Body and Wire HTTP metadata preserve actual wire evidence. Unknown/malformed charset and bare UTF16 without BOM keep Unsupported cause; normalized malformed JSON retains SyntaxError and original bytes. Existing RemoveInterfaceRecord and generic routerJSONFromResponse charset/status/body defaults stay unchanged.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveExplicitCharsetsKeepRawProofAndLegacy`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveCharsetFailuresKeepOriginalResponse`.
- The existing cloud Service exposes successful None as Value null after charset-aware parsing while diagnostic ResponseJSON and optional normalized object Wire preserve parsed fields; actual raw Envelope/Header/status remain independent. Unsupported charset does not expose successful Value/ResponseJSON and retains original accepted ResponseError proof.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveExplicitCharsetReceiptAndNone`.

Python/Go differences:

- Existing RemoveInterfaceRecord keeps its original parsed-JSON return. The separate Service.RemoveRouterInterface returns Source None as Value null with an SDK-owned diagnostic ResponseJSON and physical response receipt.
- The cloud remove_router_interface flow uses SDK-owned validation, options, port-first selection, a None-equivalent result and actual response evidence, without a new Connection forwarding method. conn.Network(ctx) service construction may authenticate and discover before argument validation; a missing ID on a directly selected or nil Service fails without HTTP.
- Go router input is a validated nonblank UTF8 string ID, escaping slash/percent/query as one segment and rejecting controls/traversal. Source arbitrary Router Resource/dict/null/nonstring dynamic dispatch and URI/class/member/session/cache behavior remain shared SDK-R1/C1/S1 foundation scope. No name lookup is invented, and broad Python runtime objects are not claimed from raw JSON preservation.
- Go returns an owned diagnostic result with Value:null for Source None plus actual parsed ResponseJSON/Wire/Envelope/Header/StatusCode rather than returning a bare nil. Failed observations never expose successful Value/ResponseJSON or a synthetic Router Resource. Valid JSON numeric tokens preserve Go raw precision; Python floating/runtime representation and wire key order remain explicit Go policies.
- The closed Source100..399/physical-fault callback suppression opt-in leaves existing native and legacy leaf policies unchanged. Headers are a concrete Go transport convenience with immutable selected source/client/captured framing and live token renewal. Stock Neutron27 project ADMIN_OR_PROJECT_MEMBER gate is documented without inferring parent-owner reader/shared-subnet authorization or live ACL acceptance.
- Evidence is actual CPython3.14.8 pinned cloud/Proxy/Resource/Router code with Adapter.put/_get_connection fixtures (75 observations), plus independently audited bounded router13 observations. Source/AST, Go/race, main compilation and live backend authorization are separate claims; no live backend was used.
- Cycle12 changes only the fixed cloud dependency to reuse internal/responsejson.Normalize for bounded explicit charsets. The generic/legacy decoder and common ABI/codec set remain unchanged. BOM is required for bare UTF16/32; explicit LE/BE works without it. Actual raw proof stays separate from normalized diagnostic JSON. Reused Source75 and independent13 observations are not rerun or combined into a new count. Worker ran gofmt only; root owns Go/race/main builds; no mapping promotion is claimed.
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Only standard finite JSON numbers are accepted: NaN, Infinity and overflowing literals such as 1e400 return ErrUnsupported with the original response, where Python's json loads them as floats, and a latest microversion must be written as "latest"; see Deliberate runtime differences in docs/sdk-support-ledger.md.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
attached = conn.network.add_interface_to_router("ROUTER_ID", subnet="SUBNET_ID")
detached = conn.network.remove_interface_from_router("ROUTER_ID", port="PORT_ID")

added = conn.network.update_router("ROUTER_ID", external_gateway_info={
    "network_id": "EXTERNAL_NETWORK_ID"
})
removed = conn.network.update_router("ROUTER_ID", external_gateway_info=None)

# conn.network.add_gateway_to_router("ROUTER_ID", network_id="EXTERNAL_NETWORK_ID")
# conn.network.remove_gateway_from_router("ROUTER_ID")
```

```sh
go run . -cloud dev -router ROUTER_ID -action add-interface -subnet SUBNET_ID
go run . -cloud dev -router ROUTER_ID -action remove-interface -port PORT_ID
go run . -cloud dev -router ROUTER_ID -action add-gateway -network EXTERNAL_NETWORK_ID
go run . -cloud dev -router ROUTER_ID -action remove-gateway
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("router", "", "literal router ID")
    action := flag.String("action", "", "add-interface, remove-interface, add-gateway, remove-gateway")
    port := flag.String("port", "", "interface port ID; wins subnet")
    subnet := flag.String("subnet", "", "interface subnet ID")
    network := flag.String("network", "", "single external gateway network ID")
    advertise := flag.Bool("advertise-host", false, "add-interface only; requires supporting backend")
    legacy := flag.Bool("legacy-source-route", false, "gateway only; requires backend legacy action support")
    flag.Parse()
    if *id == "" { log.Fatal("-router is required") }
    switch *action {
    case "add-interface", "remove-interface":
        if *port == "" && *subnet == "" { log.Fatal("provide -port or -subnet") }
        if *legacy { log.Fatal("-legacy-source-route is a gateway option") }
        if *action == "remove-interface" && *advertise { log.Fatal("-advertise-host is add-interface only") }
    case "add-gateway":
        if *network == "" { log.Fatal("-network is required") }
    case "remove-gateway":
        if *network != "" && !*legacy { log.Fatal("stock remove-gateway accepts no network attribute") }
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var result *routers.RouterJSONResult
    switch *action {
    case "add-interface", "remove-interface":
        options := []routers.RouterInterfaceOption{}
        if *port != "" { options = append(options, routers.WithRouterInterfacePort(*port)) }
        if *subnet != "" { options = append(options, routers.WithRouterInterfaceSubnet(*subnet)) }
        if *action == "add-interface" {
            options = append(options, routers.WithRouterInterfaceAdvertiseHost(*advertise))
            result, err = service.Routers.AddInterfaceRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveInterfaceRecord(ctx, *id, options...)
        }
    case "add-gateway", "remove-gateway":
        options := []routers.RouterGatewayOption{routers.WithSourceLegacyGatewayRoutes(*legacy)}
        if *network != "" { options = append(options, routers.WithRouterGatewayField("network_id", *network)) }
        if *action == "add-gateway" {
            result, err = service.Routers.AddGatewayRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveGatewayRecord(ctx, *id, options...)
        }
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Value != nil { fmt.Printf("complete JSON value=%s\n", result.Value) }
    }
    if err != nil { log.Fatal(err) }
}
```

## References

- [L3 Db](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/l3_db.py)
- [Neutron 27.0.0 L3 extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/l3.py)
- [neutron-lib 3.21.1 ACTION_MAP](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/l3.py)
- [Router Actions Python](router-actions-python.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)
- [Install](../../docs/install.md)
