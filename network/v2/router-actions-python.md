# Router Actions Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/add_external_gateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddExternalGatewaysRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionSeed`

- One PUT uses the literal router ID and the exact pinned action route, without prefetch or envelope/alias/body default synthesis. Complete positional JSON preserves omitted/null/false/zero/empty/nested unknown values and large integers; Python None corresponds to bodyless nil while explicit RawMessage null is a documented Go input.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement`.
- The pinned Router canonical descriptor view applies response aliases in parsed JSON member order, project/tenant fallback, missing null and tags empty-list defaults, bool/int/list/dict conversions, inherited response if_match and self removal. Original Wire/Envelope/headers/status are independent and remain actual HTTP evidence.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire`.
- All five actions admit Source successful statuses 200..399 and preserve clean native HTTP errors, including strict 404. Malformed/empty JSON is tolerated, while parsed nonobjects, null router envelopes and descriptor failures carry actual response body/header/status in ResponseError.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence`.
- Accepted and rejected physical Read/Close/cancel/source-changed-and-restored-on-Close failures preserve every observed cause and HTTP body/header/status. A faulty acknowledgement never becomes a complete Resource or resends, and rejection handling failures never become clean missing.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults`.
- Selected network source and input body are captured before custom options/marshalers. Header/body/retained-callback mutations cannot alter snapshots; invalid clients/routes/IDs/protected headers/options and option/marshaler context/source causes are terminal before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs`, `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords`.
- Native retries and reauthentication keep request JSON and route ownership; request-body/response-decoder/source mutations are terminal with original rejection evidence, and expanded retry OkCodes do not change the original SDK success policy.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership`.
- Optional cached Router seeds remain independent and survive empty acknowledgements without being submitted or fabricated into Wire/HTTP metadata. Response fields including explicit null overlay cached values; literal route ID owns the initial view, inherited seeded if_match survives absent response Header, and response Header overrides it. Malformed/control/colliding seed aliases fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire`.

Python/Go differences:

- Go returns new owned Resource/Wire/Envelope/header/status records and optional copied cached seeds rather than mutating a Python Router object. An explicit route ID overrides cached seed ID; returned response ID, including null, still updates the view. Connection-derived location is null and remains tracked in the shared Resource/cache/session goal.
- The caller supplies the complete action JSON; these direct Source actions do not use ordinary Neutron attribute mutation aliases, revision options or implicit router envelope defaults. Go headers and cached-seed options are explicit conveniences without auth/service/route overrides.
- JSON member order preserves response canonical/wire alias precedence. A Go map seed with conflicting aliases is rejected instead of depending on iteration order. Inputs use complete UTF-8 JSON and ordinary Python JSON-domain projections; arbitrary Python objects, input Router mutation and all Python/Requests runtime behavior are not claimed as complete SDK parity.
- Integer descriptor Unicode digit tables use the shared audited Unicode 16.0/CPython 3.14.8 profile and arbitrary precision, rather than a configurable Python runtime string limit. Bool decimal/exponent tokens follow ordinary Python binary64 zero/nonzero behavior, including underflow/overflow, while Wire retains their exact JSON bytes.
- Stock Neutron 27.0.0 permits all five actions for admin or member of the owning project. add/update external_gateways enable_snat and external_fixed_ips are separate admin-only attribute policies; the SDK preserves supplied values and native server authorization errors but adds no separate admin workflow.

### `python:network/v2/add_extra_routes_to_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddExtraRoutesRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionSeed`

- One PUT uses the literal router ID and the exact pinned action route, without prefetch or envelope/alias/body default synthesis. Complete positional JSON preserves omitted/null/false/zero/empty/nested unknown values and large integers; Python None corresponds to bodyless nil while explicit RawMessage null is a documented Go input.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement`.
- The pinned Router canonical descriptor view applies response aliases in parsed JSON member order, project/tenant fallback, missing null and tags empty-list defaults, bool/int/list/dict conversions, inherited response if_match and self removal. Original Wire/Envelope/headers/status are independent and remain actual HTTP evidence.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire`.
- All five actions admit Source successful statuses 200..399 and preserve clean native HTTP errors, including strict 404. Malformed/empty JSON is tolerated, while parsed nonobjects, null router envelopes and descriptor failures carry actual response body/header/status in ResponseError.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence`.
- Accepted and rejected physical Read/Close/cancel/source-changed-and-restored-on-Close failures preserve every observed cause and HTTP body/header/status. A faulty acknowledgement never becomes a complete Resource or resends, and rejection handling failures never become clean missing.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults`.
- Selected network source and input body are captured before custom options/marshalers. Header/body/retained-callback mutations cannot alter snapshots; invalid clients/routes/IDs/protected headers/options and option/marshaler context/source causes are terminal before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs`, `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords`.
- Native retries and reauthentication keep request JSON and route ownership; request-body/response-decoder/source mutations are terminal with original rejection evidence, and expanded retry OkCodes do not change the original SDK success policy.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership`.
- Optional cached Router seeds remain independent and survive empty acknowledgements without being submitted or fabricated into Wire/HTTP metadata. Response fields including explicit null overlay cached values; literal route ID owns the initial view, inherited seeded if_match survives absent response Header, and response Header overrides it. Malformed/control/colliding seed aliases fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire`.

Python/Go differences:

- Go returns new owned Resource/Wire/Envelope/header/status records and optional copied cached seeds rather than mutating a Python Router object. An explicit route ID overrides cached seed ID; returned response ID, including null, still updates the view. Connection-derived location is null and remains tracked in the shared Resource/cache/session goal.
- The caller supplies the complete action JSON; these direct Source actions do not use ordinary Neutron attribute mutation aliases, revision options or implicit router envelope defaults. Go headers and cached-seed options are explicit conveniences without auth/service/route overrides.
- JSON member order preserves response canonical/wire alias precedence. A Go map seed with conflicting aliases is rejected instead of depending on iteration order. Inputs use complete UTF-8 JSON and ordinary Python JSON-domain projections; arbitrary Python objects, input Router mutation and all Python/Requests runtime behavior are not claimed as complete SDK parity.
- Integer descriptor Unicode digit tables use the shared audited Unicode 16.0/CPython 3.14.8 profile and arbitrary precision, rather than a configurable Python runtime string limit. Bool decimal/exponent tokens follow ordinary Python binary64 zero/nonzero behavior, including underflow/overflow, while Wire retains their exact JSON bytes.
- Stock Neutron 27.0.0 permits all five actions for admin or member of the owning project. add/update external_gateways enable_snat and external_fixed_ips are separate admin-only attribute policies; the SDK preserves supplied values and native server authorization errors but adds no separate admin workflow.

### `python:network/v2/remove_external_gateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveExternalGatewaysRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionSeed`

- One PUT uses the literal router ID and the exact pinned action route, without prefetch or envelope/alias/body default synthesis. Complete positional JSON preserves omitted/null/false/zero/empty/nested unknown values and large integers; Python None corresponds to bodyless nil while explicit RawMessage null is a documented Go input.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement`.
- The pinned Router canonical descriptor view applies response aliases in parsed JSON member order, project/tenant fallback, missing null and tags empty-list defaults, bool/int/list/dict conversions, inherited response if_match and self removal. Original Wire/Envelope/headers/status are independent and remain actual HTTP evidence.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire`.
- All five actions admit Source successful statuses 200..399 and preserve clean native HTTP errors, including strict 404. Malformed/empty JSON is tolerated, while parsed nonobjects, null router envelopes and descriptor failures carry actual response body/header/status in ResponseError.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence`.
- Accepted and rejected physical Read/Close/cancel/source-changed-and-restored-on-Close failures preserve every observed cause and HTTP body/header/status. A faulty acknowledgement never becomes a complete Resource or resends, and rejection handling failures never become clean missing.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults`.
- Selected network source and input body are captured before custom options/marshalers. Header/body/retained-callback mutations cannot alter snapshots; invalid clients/routes/IDs/protected headers/options and option/marshaler context/source causes are terminal before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs`, `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords`.
- Native retries and reauthentication keep request JSON and route ownership; request-body/response-decoder/source mutations are terminal with original rejection evidence, and expanded retry OkCodes do not change the original SDK success policy.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership`.
- Optional cached Router seeds remain independent and survive empty acknowledgements without being submitted or fabricated into Wire/HTTP metadata. Response fields including explicit null overlay cached values; literal route ID owns the initial view, inherited seeded if_match survives absent response Header, and response Header overrides it. Malformed/control/colliding seed aliases fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire`.

Python/Go differences:

- Go returns new owned Resource/Wire/Envelope/header/status records and optional copied cached seeds rather than mutating a Python Router object. An explicit route ID overrides cached seed ID; returned response ID, including null, still updates the view. Connection-derived location is null and remains tracked in the shared Resource/cache/session goal.
- The caller supplies the complete action JSON; these direct Source actions do not use ordinary Neutron attribute mutation aliases, revision options or implicit router envelope defaults. Go headers and cached-seed options are explicit conveniences without auth/service/route overrides.
- JSON member order preserves response canonical/wire alias precedence. A Go map seed with conflicting aliases is rejected instead of depending on iteration order. Inputs use complete UTF-8 JSON and ordinary Python JSON-domain projections; arbitrary Python objects, input Router mutation and all Python/Requests runtime behavior are not claimed as complete SDK parity.
- Integer descriptor Unicode digit tables use the shared audited Unicode 16.0/CPython 3.14.8 profile and arbitrary precision, rather than a configurable Python runtime string limit. Bool decimal/exponent tokens follow ordinary Python binary64 zero/nonzero behavior, including underflow/overflow, while Wire retains their exact JSON bytes.
- Stock Neutron 27.0.0 permits all five actions for admin or member of the owning project. add/update external_gateways enable_snat and external_fixed_ips are separate admin-only attribute policies; the SDK preserves supplied values and native server authorization errors but adds no separate admin workflow.

### `python:network/v2/remove_extra_routes_from_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveExtraRoutesRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionSeed`

- One PUT uses the literal router ID and the exact pinned action route, without prefetch or envelope/alias/body default synthesis. Complete positional JSON preserves omitted/null/false/zero/empty/nested unknown values and large integers; Python None corresponds to bodyless nil while explicit RawMessage null is a documented Go input.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement`.
- The pinned Router canonical descriptor view applies response aliases in parsed JSON member order, project/tenant fallback, missing null and tags empty-list defaults, bool/int/list/dict conversions, inherited response if_match and self removal. Original Wire/Envelope/headers/status are independent and remain actual HTTP evidence.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire`.
- All five actions admit Source successful statuses 200..399 and preserve clean native HTTP errors, including strict 404. Malformed/empty JSON is tolerated, while parsed nonobjects, null router envelopes and descriptor failures carry actual response body/header/status in ResponseError.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence`.
- Accepted and rejected physical Read/Close/cancel/source-changed-and-restored-on-Close failures preserve every observed cause and HTTP body/header/status. A faulty acknowledgement never becomes a complete Resource or resends, and rejection handling failures never become clean missing.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults`.
- Selected network source and input body are captured before custom options/marshalers. Header/body/retained-callback mutations cannot alter snapshots; invalid clients/routes/IDs/protected headers/options and option/marshaler context/source causes are terminal before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs`, `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords`.
- Native retries and reauthentication keep request JSON and route ownership; request-body/response-decoder/source mutations are terminal with original rejection evidence, and expanded retry OkCodes do not change the original SDK success policy.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership`.
- Optional cached Router seeds remain independent and survive empty acknowledgements without being submitted or fabricated into Wire/HTTP metadata. Response fields including explicit null overlay cached values; literal route ID owns the initial view, inherited seeded if_match survives absent response Header, and response Header overrides it. Malformed/control/colliding seed aliases fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire`.

Python/Go differences:

- Go returns new owned Resource/Wire/Envelope/header/status records and optional copied cached seeds rather than mutating a Python Router object. An explicit route ID overrides cached seed ID; returned response ID, including null, still updates the view. Connection-derived location is null and remains tracked in the shared Resource/cache/session goal.
- The caller supplies the complete action JSON; these direct Source actions do not use ordinary Neutron attribute mutation aliases, revision options or implicit router envelope defaults. Go headers and cached-seed options are explicit conveniences without auth/service/route overrides.
- JSON member order preserves response canonical/wire alias precedence. A Go map seed with conflicting aliases is rejected instead of depending on iteration order. Inputs use complete UTF-8 JSON and ordinary Python JSON-domain projections; arbitrary Python objects, input Router mutation and all Python/Requests runtime behavior are not claimed as complete SDK parity.
- Integer descriptor Unicode digit tables use the shared audited Unicode 16.0/CPython 3.14.8 profile and arbitrary precision, rather than a configurable Python runtime string limit. Bool decimal/exponent tokens follow ordinary Python binary64 zero/nonzero behavior, including underflow/overflow, while Wire retains their exact JSON bytes.
- Stock Neutron 27.0.0 permits all five actions for admin or member of the owning project. add/update external_gateways enable_snat and external_fixed_ips are separate admin-only attribute policies; the SDK preserves supplied values and native server authorization errors but adds no separate admin workflow.

### `python:network/v2/update_external_gateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.UpdateExternalGatewaysRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterActionSeed`

- One PUT uses the literal router ID and the exact pinned action route, without prefetch or envelope/alias/body default synthesis. Complete positional JSON preserves omitted/null/false/zero/empty/nested unknown values and large integers; Python None corresponds to bodyless nil while explicit RawMessage null is a documented Go input.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement`.
- The pinned Router canonical descriptor view applies response aliases in parsed JSON member order, project/tenant fallback, missing null and tags empty-list defaults, bool/int/list/dict conversions, inherited response if_match and self removal. Original Wire/Envelope/headers/status are independent and remain actual HTTP evidence.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire`.
- All five actions admit Source successful statuses 200..399 and preserve clean native HTTP errors, including strict 404. Malformed/empty JSON is tolerated, while parsed nonobjects, null router envelopes and descriptor failures carry actual response body/header/status in ResponseError.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence`.
- Accepted and rejected physical Read/Close/cancel/source-changed-and-restored-on-Close failures preserve every observed cause and HTTP body/header/status. A faulty acknowledgement never becomes a complete Resource or resends, and rejection handling failures never become clean missing.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults`.
- Selected network source and input body are captured before custom options/marshalers. Header/body/retained-callback mutations cannot alter snapshots; invalid clients/routes/IDs/protected headers/options and option/marshaler context/source causes are terminal before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs`, `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords`.
- Native retries and reauthentication keep request JSON and route ownership; request-body/response-decoder/source mutations are terminal with original rejection evidence, and expanded retry OkCodes do not change the original SDK success policy.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership`.
- Optional cached Router seeds remain independent and survive empty acknowledgements without being submitted or fabricated into Wire/HTTP metadata. Response fields including explicit null overlay cached values; literal route ID owns the initial view, inherited seeded if_match survives absent response Header, and response Header overrides it. Malformed/control/colliding seed aliases fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_actions_contracts_test.go:TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire`.

Python/Go differences:

- Go returns new owned Resource/Wire/Envelope/header/status records and optional copied cached seeds rather than mutating a Python Router object. An explicit route ID overrides cached seed ID; returned response ID, including null, still updates the view. Connection-derived location is null and remains tracked in the shared Resource/cache/session goal.
- The caller supplies the complete action JSON; these direct Source actions do not use ordinary Neutron attribute mutation aliases, revision options or implicit router envelope defaults. Go headers and cached-seed options are explicit conveniences without auth/service/route overrides.
- JSON member order preserves response canonical/wire alias precedence. A Go map seed with conflicting aliases is rejected instead of depending on iteration order. Inputs use complete UTF-8 JSON and ordinary Python JSON-domain projections; arbitrary Python objects, input Router mutation and all Python/Requests runtime behavior are not claimed as complete SDK parity.
- Integer descriptor Unicode digit tables use the shared audited Unicode 16.0/CPython 3.14.8 profile and arbitrary precision, rather than a configurable Python runtime string limit. Bool decimal/exponent tokens follow ordinary Python binary64 zero/nonzero behavior, including underflow/overflow, while Wire retains their exact JSON bytes.
- Stock Neutron 27.0.0 permits all five actions for admin or member of the owning project. add/update external_gateways enable_snat and external_fixed_ips are separate admin-only attribute policies; the SDK preserves supplied values and native server authorization errors but adds no separate admin workflow.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
router_id = "ROUTER_ID"
route_body = {"router": {"routes": [
    {"destination": "192.0.2.0/24", "nexthop": "10.0.0.9"}
]}}
gateway_body = {"router": {"external_gateways": [
    {"network_id": "EXTERNAL_NETWORK_ID"}
]}}

added = conn.network.add_extra_routes_to_router(router_id, route_body)
removed = conn.network.remove_extra_routes_from_router(router_id, route_body)
gateway_added = conn.network.add_external_gateways(router_id, gateway_body)
gateway_updated = conn.network.update_external_gateways(router_id, gateway_body)
gateway_removed = conn.network.remove_external_gateways(router_id, gateway_body)
print(added.to_dict())
```

```sh
go run . -cloud dev -router ROUTER_ID -action add-routes \
  -body '{"router":{"routes":[{"destination":"192.0.2.0/24","nexthop":"10.0.0.9"}]}}'
go run . -cloud dev -router ROUTER_ID -action add-gateways \
  -body '{"router":{"external_gateways":[{"network_id":"EXTERNAL_NETWORK_ID"}]}}'
```

```go
package main

import (
    "context"
    "encoding/json"
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
    action := flag.String("action", "", "add-routes, remove-routes, add-gateways, update-gateways, remove-gateways")
    body := flag.String("body", "", "complete JSON action body, including router envelope")
    flag.Parse()
    if *id == "" || !json.Valid([]byte(*body)) { log.Fatal("-router and a valid JSON -body are required") }
    switch *action {
    case "add-routes", "remove-routes", "add-gateways", "update-gateways", "remove-gateways":
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Routers
    raw := json.RawMessage(*body)
    var result *routers.RouterActionRecord
    switch *action {
    case "add-routes": result, err = api.AddExtraRoutesRecord(ctx, *id, raw)
    case "remove-routes": result, err = api.RemoveExtraRoutesRecord(ctx, *id, raw)
    case "add-gateways": result, err = api.AddExternalGatewaysRecord(ctx, *id, raw)
    case "update-gateways": result, err = api.UpdateExternalGatewaysRecord(ctx, *id, raw)
    case "remove-gateways": result, err = api.RemoveExternalGatewaysRecord(ctx, *id, raw)
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Resource != nil {
            view, encodeErr := json.MarshalIndent(result.Resource, "", "  ")
            if encodeErr != nil { log.Fatal(encodeErr) }
            fmt.Printf("Source descriptor view=%s\n", view)
        }
    }
    if err != nil { log.Fatal(err) }
}
```

```go
cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
    "name": json.RawMessage(`"cached-edge"`),
    "routes": json.RawMessage(`[]`),
}}}
row, err := api.AddExtraRoutesRecord(ctx, "ROUTER_ID", json.RawMessage(`{"router":{"routes":[]}}`),
    routers.WithRouterActionSeed(cached),
    routers.WithRouterActionHeaders(map[string]string{"X-Client-Trace": "router-action"}))
```

## References

- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)
- [Install](../../docs/install.md)
- [Implementation Plan](../../docs/implementation-plan.md)
