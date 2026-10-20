# Acceleratorrequests

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:accelerator/v2/accelerator_requests`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`

- ARQ WithStatus(bound) filters state locally and forwards explicit instance=instance, without manufacturing a status query. Explicit status=Bound still reaches the server and preserves its HTTP 400; caller limit or marker options are rejected.
  Tests: `api/accelerator_status_filter_test.go:TestAcceleratorStatusFilterStaysLocal`.
- ARQ Get and All reject absent/empty/comma-containing response identities rather than returning an unusable request.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestReadRejectsMissingIdentity`.

Python/Go differences:

- Python models inherit QueryParameters(limit, marker); Resource.list filters declared Body attributes locally and omits other unknown parameters. Go WithQuery forwards explicit query values to Cyborg, with WithName/WithStatus providing explicit local filters where supported; it rejects caller-supplied limit/marker and follows only server continuation links.
- Pinned Resource.list omits unknown bind_state/instance queries; Go explicitly forwards those controller filters and treats ARQ state as a local status mapping. The controller can return 423 for an instance whose ARQs are not all resolved.

Unresolved boundaries:

- Audit and document the whole inherited Python Proxy._list/Resource.list contract: body-attribute local equality filtering, unknown query omission, paginated/base_path/max_items/jmespath arguments, limit/marker fallback, Link-header continuation, URI rewriting, lazy iteration and Adapter error/microversion behavior. Current fixtures cover the stated Go subset and do not establish those inherited contracts.
- Add direct ARQ list bind_state=resolved and combined instance/bind_state 423 observations, metadata/null fields, successful multiple items across continuation, empty linked page and break/cycle behavior before claiming the whole list operation.

### `python:accelerator/v2/create_accelerator_request`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.WithCreateHeader`

- POST /accelerator_requests sends flat device_profile_name and explicit extension/header values. A 201 arqs array preserves both created ARQs, raw large numbers, null versus empty hostname and unknown envelope fields; envelope/item headers are independent clones.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestCreateRetainsBatch`.
- Accepted empty batches are non-nil; bad items return the accepted response and every valid item including items after an invalid entry, with raw original response bytes and metadata. Malformed/root/envelope/identity failures issue one POST; nonaccepted 403 preserves the actual HTTP cause.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestCreatePartialAndErrors`.
- Cancellation only after accepted 201 headers and partial body reading retains status/request-id/raw prefix plus context.Canceled, closes the response body once and bypasses RetryFunc. Malformed accepted JSON retains original whitespace and syntax cause without calling the configured retry hook.
  Tests: `api/accelerator_request_response_boundary_test.go:TestAcceleratorRequestAcceptedBodyCancellationRetainsResponse`, `api/accelerator_request_response_boundary_test.go:TestAcceleratorRequestMalformedAcceptedJSONBypassesRetryHook`.
- Missing device profile, protected body/header fields, nil/query/argument options and canceled context fail before HTTP.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestPreflight`.

Python/Go differences:

- Pinned AcceleratorRequest.create_opts removes the request envelope, but _consume_attrs replaces any arqs response with arqs[0], discarding later created resources and failing on an empty array. Go deliberately returns the entire batch including an empty batch and partial successful items together with a decode/read error.
- Go exposes RawBody and accepted HTTP metadata without cleaning up or reissuing accepted creation on decoder/read failure; it does not emulate Python mutable Resource initialization/response-state updates. Existing provider transport/auth retry hooks remain configured.

Unresolved boundaries:

- Audit Python Proxy._create, Resource initialization/dirty and unknown attributes, create_opts, _transform_create_request, _translate_response and Adapter error/retry/microversion behavior before treating the entire Python create operation as mapped. Concrete body/response tests do not validate that inherited pipeline.
- The explicit batch/read-failure policy is documented and well tested, but Python initialization/_consume_attrs mappings, body aliases/unknown attributes, response-envelope conversion and Adapter state/exception semantics still require a whole-operation audit rather than completion inferred from Create existence.

### `python:accelerator/v2/delete_accelerator_request`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.DeleteMany`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.DeleteByInstance`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.WithDeleteHeader`

- Single DELETE uses the collection query arqs=a1 with no body. Missing single requests are ignored by default or return ErrNotFound/HTTP404 under WithMissingError; explicit batch arqs=a1,a2 preserves strict 404 and the service-token header.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestDeleteSelectorsAndPolicy`.
- DeleteByInstance sends only instance=instance plus the caller service token, returns 204 twice under the idempotent fixture, and rejects nil/duplicate/comma/empty IDs before further HTTP.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestDeleteSelectorsAndPolicy`.

Python/Go differences:

- Pinned inherited Resource.delete appends an ID to /accelerator_requests; the controller instead requires DELETE /accelerator_requests?arqs=uuid[,uuid] or ?instance=uuid. Go deliberately corrects this collection endpoint and exposes explicit single, batch and instance selectors.
- Single deletion retains default ignore-missing; DeleteMany intentionally keeps 404 strict so partial batch non-deletion is observable. A caller may supply X-Service-Token, while the SDK does not fabricate a valid service identity or client-side authorization result.

Unresolved boundaries:

- Audit Python Proxy._delete/_get_resource and Resource._raw_delete/_translate_response/Adapter behavior for Resource/dict inputs, alternate identity, response-state mutation, cache, microversion and exception classification. UUID DELETE and ignore-missing fixtures cover only the stated Go policy.
- Audit and document Python member-URL correction versus inherited Resource deletion response-state behavior; add DELETE-specific 403/service-token failure, cancellation, strict 204/other-code and bound/unbound selector observations. The fixture observes header transmission, not Cyborg token validation or authorization.

### `python:accelerator/v2/get_accelerator_request`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Get`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`

- ARQ GET decodes null instance_uuid and attach_handle_info, and the state waiter polls /accelerator_requests/a1 from Initial to Bound; names and comma-containing single IDs are unsupported/invalid.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestReadAndStatePolicy`.
- GET rejects responses with missing, empty or comma-containing UUID, including the matching list decode boundary.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestReadRejectsMissingIdentity`.

Python/Go differences:

- Python deprecated fields only emits a warning and never changes the GET; Go omits that argument and returns its own typed value with raw object JSON and HTTP metadata, without mutable Resource/cache semantics.
- Go UUID identity is distinct from repeated device_profile_name and uses state for its common status mapping. Singletons require exactly one object rather than accepting the first of an arbitrary arqs response through Python _consume_attrs.

Unresolved boundaries:

- Audit Python Proxy._get/_get_resource, Resource.fetch/_translate_response and Adapter behavior for string, existing Resource and dict inputs, response conversion/cache, alternate identity, default microversion and exception classification. The HTTP fixture establishes the Go request/result policy, not complete inherited Python equivalence.
- Add getter-specific 404/403/metadata/header/cancellation and singular/plural-singleton response observations. The state-wait and missing-identity fixtures do not independently establish all get behavior or the inherited Python first-item decoding policy.

### `python:accelerator/v2/patch_accelerator_request`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Patch`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.PatchMany`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Bind`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Unbind`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.WithBindingHeader`

- Bind sends a UUID-keyed object with hostname/device_rp_uuid/instance_uuid and optional project_id to collection PATCH /accelerator_requests, with accelerator 2.10 and X-Service-Token. Unbind sends remove operations without values; 202 yields HTTP metadata and no invented resource body.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingCollectionURL`.
- PatchMany snapshots two UUID-keyed patches and pointer values before a 401 reauthentication callback mutates/deletes original inputs; the second attempt still contains both original documents.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingSnapshotAndBatch`.
- Unknown/replace/mixed/duplicate/incomplete binding operations, nil patches, names, protected/capability options and project_id below 2.1 fail before HTTP; an actual missing-service-token 403 keeps status/request-id.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingPreflightAndAuthorization`.

Python/Go differences:

- Pinned AcceleratorRequest.patch prepares a requires_id=True member URL even though its _convert_patch creates a UUID-keyed object; the Cyborg PATCH endpoint is the collection /accelerator_requests. Go corrects that endpoint and offers concrete Bind/Unbind/Patch/PatchMany documents.
- Go requires add/remove binding fields and explicitly gates project_id at selected 2.1; it returns metadata for the controller empty 202, rather than a mutable Resource. Python can short-circuit an empty clean patch, derive dirty-state patches, convert field aliases and overwrites its patch microversion argument with _get_microversion.
- The controller validates service-request authorization for bind/unbind and admin ownership for explicit project_id; a supplied header is only transmitted by Go, and server 403 remains observable. No client-side authorization success is inferred.

Unresolved boundaries:

- Audit inherited Proxy._get_resource, Resource initialization/dirty generated patch/alias conversion, _commit/_translate_response/default microversion and Adapter retry/error policy, and define the documented reduced concrete binding scope before completing the entire Python patch operation.
- Add canceled binding and strict expected-code/response-body observations and verify controller 2.1 project ownership/server authorization semantics; tests verify selected-version preflight and header/403 propagation, not real service-token/admin validation.

### `python:accelerator/v2/update_accelerator_request`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Patch`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Bind`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests.API.Unbind`

- Bind sends a UUID-keyed object with hostname/device_rp_uuid/instance_uuid and optional project_id to collection PATCH /accelerator_requests, with accelerator 2.10 and X-Service-Token. Unbind sends remove operations without values; 202 yields HTTP metadata and no invented resource body.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingCollectionURL`.
- PatchMany snapshots two UUID-keyed patches and pointer values before a 401 reauthentication callback mutates/deletes original inputs; the second attempt still contains both original documents.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingSnapshotAndBatch`.
- Unknown/replace/mixed/duplicate/incomplete binding operations, nil patches, names, protected/capability options and project_id below 2.1 fail before HTTP; an actual missing-service-token 403 keeps status/request-id.
  Tests: `api/accelerator_request_binding_test.go:TestAcceleratorRequestBindingPreflightAndAuthorization`.

Python/Go differences:

- Unlike broken update_deployable, pinned deprecated update_accelerator_request emits a deprecation warning and delegates patch_accelerator_request(uuid, properties). Go exposes the current explicit binding patch API and omits this deprecated alias/warning.
- Pinned AcceleratorRequest.patch prepares a requires_id=True member URL even though its _convert_patch creates a UUID-keyed object; the Cyborg PATCH endpoint is the collection /accelerator_requests. Go corrects that endpoint and offers concrete Bind/Unbind/Patch/PatchMany documents.
- Go requires add/remove binding fields and explicitly gates project_id at selected 2.1; it returns metadata for the controller empty 202, rather than a mutable Resource. Python can short-circuit an empty clean patch, derive dirty-state patches, convert field aliases and overwrites its patch microversion argument with _get_microversion.
- The controller validates service-request authorization for bind/unbind and admin ownership for explicit project_id; a supplied header is only transmitted by Go, and server 403 remains observable. No client-side authorization success is inferred.

Unresolved boundaries:

- Define and document and document the deprecated alias/properties argument/warning replacement policy; the current mapping table covers patch_accelerator_request but not the separate deprecated whole-operation contract.
- Audit inherited Proxy._get_resource, Resource initialization/dirty generated patch/alias conversion, _commit/_translate_response/default microversion and Adapter retry/error policy, and define the documented reduced concrete binding scope before completing the entire Python patch operation.
- Add canceled binding and strict expected-code/response-body observations and verify controller 2.1 project ownership/server authorization semantics; tests verify selected-version preflight and header/403 propagation, not real service-token/admin validation.

### `python:accelerator/v2/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`

- ARQ Wait maps the target to case-insensitive state, observes Initial then Bound on the stable request UUID and preserves null/attach information; BindFailed returns ErrFailedState.
  Tests: `api/accelerator_requests_test.go:TestAcceleratorRequestReadAndStatePolicy`.
- Device status wait is unsupported without selected 2.3 even while lower-version Enable succeeds; at 2.3 a response UUID different from the requested ID is returned without rewriting the requested identity. Attributes/profiles/deployables have no status waiter.
  Tests: `api/accelerator_boundaries_test.go:TestAcceleratorActionAndStatusVersionsAreSeparate`, `api/accelerator_read_test.go:TestAcceleratorReadFailuresAndIdentity`, `api/accelerator_read_test.go:TestAcceleratorReadResourcesAndMetadata`, `api/accelerator_profiles_test.go:TestAcceleratorProfileExactNamesAndMetadata`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`.
- Shared Go Wait policy supports exact replacement/disabled failure states, model-owned string attribute selection, unlimited wait with parent cancellation, and integer progress callbacks on initial and later nonterminal HTTP results; target/failure/HTTP results omit callbacks.
  Tests: `resource/wait_policy_test.go:TestWaitFailurePolicyReplacementAndTargetPrecedence`, `resource/wait_policy_test.go:TestWaitUnlimitedAndBoundedOptionsPreserveParentContext`, `resource/wait_attributes_test.go:TestWaitStatusAttributeSelectsNativeNodeFields`, `resource/wait_attributes_test.go:TestWaitProgressIncludesEachNonterminalNativeServerResponse`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`.

Python/Go differences:

- Python accepts an existing Resource and can return its cached target status without fetching. Go Collection.Wait always starts with a Ref lookup/HTTP result, fixes the ID once and uses five-minute/two-second defaults with context deadlines. WithStatusAttribute selects an exported string/nullable-string model field by JSON tag or Go name; WithFailureStates replaces the service predicate, WithProgressCallback reports integer progress and WithUnlimitedWait removes only the SDK timeout.

Unresolved boundaries:

- Audit cached initial Resource short-circuit, None target status, descriptor/unknown-field attribute semantics, skip_cache/fetch errors and Resource identity mutation against the documented Go model/Ref policy. The shared waiter options are implemented, but this whole inherited Python operation remains unresolved.
- Add Cyborg Device/ARQ-specific selector, callback, timeout/in-flight cancellation and configured-failure observations; native Node/Server shared-policy fixtures do not independently establish every Cyborg inherited contract.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
created, err := accel.AcceleratorRequests.Create(ctx,
    acceleratorrequests.CreateOpts{DeviceProfileName: "fpga-profile"},
    acceleratorrequests.WithCreateHeader("X-Custom", "example"))
if err != nil { return err }
for _, arq := range created.Requests {
    fmt.Println(arq.UUID, arq.State, arq.DeviceProfileGroupID)
}
```

## References

- [Cyborg API reference](https://docs.openstack.org/api-ref/accelerator/)
- [Arqs](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/arqs.py)
- [Accelerator Request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/accelerator_request.py)
- [..](../README.md)
