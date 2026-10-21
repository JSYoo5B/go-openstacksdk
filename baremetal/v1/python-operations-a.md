# Python Bare Metal Operation Reviews (part A)

This guide records, per Python openstacksdk bare metal operation, how the Go API compares with the pinned Python behavior. Status values follow the [support ledger](../../docs/sdk-support-ledger.md): `go_mapping` means the Go API and its cited tests reproduce the listed contracts with the listed deliberate differences; `unresolved` means required behavior remains unmapped or untested.

Shared differences: Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically.

Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry.

Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

## `python:baremetal/v1/add_node_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.AddTrait`

- PUT /nodes/{id}/traits/{trait} sends no body and path-escapes the node and trait.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicATraitAddSendsBodylessPutAndEscapesTrait`
- Only 204 succeeds; empty node or trait fails before HTTP; cancellation makes no request.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicATraitAddSendsBodylessPutAndEscapesTrait`, `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleAndTraitRejectEmptyInputAndHonorCancellation`

Python/Go differences:

- Python also refreshes the local Resource traits list; Go has no Resource object and returns only an error, so callers re-read the node. Python picks microversion 1.37 or the server maximum; Go sends only the configured header. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/add_runbook_trait`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no runbook API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: PUT /runbooks/{id}/traits/{trait} with a bodyless request.

## `python:baremetal/v1/allocations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.List`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.All`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.WithListQuery`

- GET /allocations sends node, resource_class, state, extension queries, comma-separated fields, limit and sort keys and follows links.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationListFiltersFollowLinksAndExtensionQuery`
- Collection All honors single-page and max-items controls; nil options fail before HTTP; cancellation makes no request.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationCollectionAllHonorsSinglePageAndMaxItems`, `api/ironic_allocations_test.go:TestIronicAllocationRejectsInvalidInputBeforeHTTP`, `api/ironic_allocations_test.go:TestIronicAllocationCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python is a lazy generator that rejects unknown query keys and caps the microversion at 1.60; Go is an iter.Seq2 and accepts explicit extension queries (the owner filter has no typed option; it can only be sent as a generic extension query, which is not separately tested for that key). Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically.

## `python:baremetal/v1/attach_vif_to_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.AttachVirtualInterface`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithAttachVirtualInterfaceOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithAttachVirtualInterfaceField`

- POST /nodes/{id}/vifs sends id with optional port_uuid or portgroup_uuid plus explicit extension fields.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualInterfacesListAttachDetach`
- port_uuid together with portgroup_uuid, a missing id, protected-field collisions and nil options fail before HTTP.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualInterfaceRejectsConflictsBeforeHTTP`
- Only 204 succeeds; cancellation makes no request.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`, `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python raises InvalidRequest locally for both port and port_group and retries 409/503 (retry_on_conflict=True by default); Go returns a validation error and never retries automatically, so the retry_on_conflict flag has no Go counterpart by design. Python raises NotSupported below microversion 1.28/1.67; Go sends only the configured header. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/attach_vmedia_to_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.AttachVirtualMedia`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithAttachVirtualMediaOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithAttachVirtualMediaField`

- POST /nodes/{id}/vmedia sends device_type and image_url, includes image_download_source only when set, and keeps explicit extension fields.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualMediaAttachDetachAndGetKeepsRawBody`
- Protected-field collisions and nil options fail before HTTP; only 204 succeeds; cancellation makes no request.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualMediaRejectsProtectedAndNilOptionsBeforeHTTP`, `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`, `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python retries 409/503 (retry_on_conflict=True by default); Go never retries automatically, so the retry_on_conflict flag has no Go counterpart by design. Python raises NotSupported below microversion 1.89; Go sends only the configured header. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/call_driver_vendor_passthru`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.API.CallVendorPassthru`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.WithCallVendorPassthruOptions`

- The caller verb is used with an escaped method query; the JSON body keeps false/empty/null values; 200/202/204 succeed and an empty body yields a nil map.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverCallVendorPassthruSendsMethodQueryAndBody`
- A missing method and nil options fail before HTTP; the method is URL-escaped; cancellation makes no request.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverCallVendorPassthruEscapesMethodAndRejectsInvalidInputBeforeHTTP`, `api/ironic_drivers_conductors_test.go:TestIronicDriverCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python first GETs the driver (get_driver) and raises ValueError for verbs outside GET/POST/PUT/DELETE; Go makes the single passthru call and forwards the verb. Python returns the raw Response; Go returns the decoded map. Passthru is never replayed. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/call_node_vendor_passthru`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.CallVendorPassthru`

- The caller verb is used with an escaped method query and the JSON body keeps false/empty/null values; an empty response yields an empty map.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVendorPassthruMethodsAndCallBodyAndStatus`
- A missing method or nil option fails before HTTP; 200/202/204 succeed and others keep the native error; cancellation makes no request.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVendorPassthruRequiresMethodAndRejectsNilOptionBeforeHTTP`, `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`, `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`
- A rejected 503 is not retried and a caller verb outside GET/POST/PUT/DELETE is forwarded as given.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAVendorPassthruForwardsCallerVerbWithoutReplay`

Python/Go differences:

- Python raises ValueError for verbs outside GET/POST/PUT/DELETE and returns the raw requests.Response; Go forwards the verb to the server and returns the decoded JSON map. Passthru may mutate remote state and is never replayed. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/chassis`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no chassis API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: a lazy generator over GET /chassis (details flag and query validation).

## `python:baremetal/v1/conductors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors.WithListQuery`

- GET /conductors sends detail, comma-separated fields, limit and sort queries plus extension queries and follows conductor_links.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicConductorListFiltersFollowLinksAndRejectDetailWithFields`
- Detail combined with fields and nil options fail before HTTP; cancellation makes no request.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicConductorListFiltersFollowLinksAndRejectDetailWithFields`, `api/ironic_drivers_conductors_test.go:TestIronicConductorCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python rejects unknown query keys and sends microversion capped at 1.49; Go accepts explicit extension queries and sends only the configured header (conductors need 1.49 or later). Go additionally rejects detail together with fields locally. Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

## `python:baremetal/v1/create_allocation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.WithCreateField`

- POST /allocations sends the flat body with extension fields; a 202 acknowledgement returns the allocating state and is not completion.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationCreateSendsBodyAndReturnsAcknowledgementState`
- Empty optional fields are omitted, only 201/202 succeed, invalid input fails before HTTP and cancellation makes no request.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationCreateOmitsEmptyOptionalFieldsAndAcceptsOnly201Or202`, `api/ironic_allocations_test.go:TestIronicAllocationRejectsInvalidInputBeforeHTTP`, `api/ironic_allocations_test.go:TestIronicAllocationCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python **attrs are replaced by CreateOpts plus library-validated extension fields; wait_for_allocation has no Go equivalent in this review, so the caller polls Get or WaitFor. A failed or uncertain create is never replayed. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/create_chassis`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no chassis API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /chassis with chassis attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_deploy_template`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no deploy template API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /deploy_templates with deploy template attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_inspection_rule`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no inspection rule API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /inspection_rules with inspection rule attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_node`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithCreateField`

- POST /nodes sends a flat body, keeps explicit false and extension fields and omits unset fields; protected or nil options fail before HTTP.
  Tests: `api/ironic_nodes_crud_test.go:TestIronicNodeCreateSendsFlatBodyKeepsFalseAndExtensions`, `api/ironic_nodes_crud_test.go:TestIronicNodeCreateOmitsUnsetOptionalFields`, `api/ironic_nodes_crud_test.go:TestIronicNodeRejectsProtectedOrNilOptionsBeforeHTTP`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python Node.create validates provision_state (enroll, manageable or available), selects the microversion required for it (1.11, 1.4 or at most 1.10) and, for manageable, issues a follow-up set_provision_state(manage, wait=True); the Go Create sends the flat body with no such preflight or follow-up and has no tests for these behaviors.
- Python caps microversion negotiation at 1.109; the Go client sends only the configured header.
- No test compares the Python create body (provision_state removed from the dirty body, unknown attrs) with the Go CreateOpts body.

## `python:baremetal/v1/create_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.WithCreateField`

- POST /ports sends the flat body with explicit false values and extension fields.
  Tests: `api/ironic_ports_test.go:TestIronicPortCreateKeepsExplicitFalseAndExtensionFields`
- Unset fields are omitted, only 201/202 succeed, invalid input fails before HTTP and cancellation makes no request.
  Tests: `api/ironic_ports_test.go:TestIronicPortCreateOmitsUnsetFieldsAndAcceptsOnly201Or202`, `api/ironic_ports_test.go:TestIronicPortRejectsInvalidInputBeforeHTTP`, `api/ironic_ports_test.go:TestIronicPortCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python **attrs are replaced by CreateOpts plus library-validated extension fields. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/create_port_group`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no port group API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /portgroups with port group attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_runbook`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no runbook API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /runbooks with runbook attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_volume_connector`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume connector API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /volume/connectors with volume connector attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/create_volume_target`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume target API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: POST /volume/targets with volume target attributes (Python **attrs) and the resource class microversion cap.

## `python:baremetal/v1/delete_allocation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.Delete`

- Remove deletes an ID reference with DELETE /allocations/{id}; 404 is ignored by default, WithMissingError reports ErrNotFound, and other failures are neither ignored nor replayed.
  Tests: `api/py_ironic_a_remove_test.go:TestPyIronicARemoveMissingPolicyMatchesIgnoreMissing`, `api/ironic_allocations_test.go:TestIronicAllocationRemoveIgnoresMissingByDefaultAndCanRequireIt`
- API.Delete accepts 202/204 only and keeps a 404 as an unexpected-status error; cancellation makes no request.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationGetAndDeleteStatusPolicies`, `api/ironic_allocations_test.go:TestIronicAllocationCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns the deleted Resource or None; Go returns only an error. A 202/204 acknowledgement is not proof the allocation is gone. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/delete_chassis`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no chassis API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /chassis/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_deploy_template`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no deploy template API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /deploy_templates/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_inspection_rule`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no inspection rule API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /inspection_rules/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.Delete`

- Remove deletes an ID reference with DELETE /nodes/{id}; a 404 is ignored by default (ignore_missing=True) and WithMissingError reports ErrNotFound (ignore_missing=False); other failures such as 409 are never ignored or replayed.
  Tests: `api/py_ironic_a_remove_test.go:TestPyIronicARemoveMissingPolicyMatchesIgnoreMissing`, `api/ironic_nodes_crud_test.go:TestIronicNodeDeleteIsStrictAndRemoveIgnoresMissingByDefault`
- API.Delete is the strict native call and keeps a 404 as an unexpected-status error; exact success codes are enforced.
  Tests: `api/ironic_nodes_crud_test.go:TestIronicNodeDeleteIsStrictAndRemoveIgnoresMissingByDefault`, `api/ironic_nodes_crud_test.go:TestIronicNodeSuccessCodesAreExactForCRUD`
- Pre-canceled deletes make no request and keep their cause.
  Tests: `api/ironic_nodes_crud_test.go:TestIronicNodeCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns the deleted Resource or None; Go returns only an error. Success acknowledgement is not proof the node is gone: use WaitForDeletion. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/delete_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Delete`

- Remove deletes an ID reference with DELETE /ports/{id}; 404 is ignored by default, WithMissingError reports ErrNotFound, and other failures are neither ignored nor replayed.
  Tests: `api/py_ironic_a_remove_test.go:TestPyIronicARemoveMissingPolicyMatchesIgnoreMissing`, `api/ironic_ports_test.go:TestIronicPortGetAndDeleteStatusPolicies`
- API.Delete accepts 202/204 only and keeps a 404 as an unexpected-status error; cancellation makes no request.
  Tests: `api/ironic_ports_test.go:TestIronicPortGetAndDeleteStatusPolicies`, `api/ironic_ports_test.go:TestIronicPortCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns the deleted Resource or None; Go returns only an error. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/delete_port_group`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no port group API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /portgroups/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_runbook`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no runbook API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /runbooks/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_volume_connector`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume connector API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /volume/connectors/{id} with ignore_missing=True by default.

## `python:baremetal/v1/delete_volume_target`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume target API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: DELETE /volume/targets/{id} with ignore_missing=True by default.

## `python:baremetal/v1/deploy_templates`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no deploy template API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: a lazy generator over GET /deploy_templates (details flag and query validation).

## `python:baremetal/v1/detach_vif_from_node`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.DetachVirtualInterface`

- DELETE /nodes/{id}/vifs/{vif} sends no body, accepts only 204 and keeps a native error otherwise.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualInterfacesListAttachDetach`, `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python detach_vif_from_node(node, vif_id, ignore_missing=True) treats HTTP 400 (VIF not attached) as a successful no-op and returns False, and returns True on success; Go DetachVirtualInterface is strict, returns only an error and has no ignore-missing option or 400 handling.
- Python retries 409/503; Go never replays, which would need to be documented as a difference once the ignore-missing behavior is mapped.

## `python:baremetal/v1/detach_vmedia_from_node`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.DetachVirtualMedia`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithDetachVirtualMediaOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithDetachVirtualMediaQuery`

- DELETE /nodes/{id}/vmedia sends the comma-separated device_types query when set, accepts only 204 and cancellation makes no request.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeVirtualMediaAttachDetachAndGetKeepsRawBody`, `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`, `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python sends the device types as a DELETE JSON body {"device_types": "a,b"} while the native Go call sends them as a device_types query parameter; the server-side equivalence of the two wire forms is not verified, so the request contract is unmatched.
- Python logs a debug message for HTTP 400 (no virtual media) and still raises the response error; the Go 400 outcome is not tested.
- Python retries 409/503; Go never replays.

## `python:baremetal/v1/disable_node_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.SetConsoleMode`

- PUT /nodes/{id}/states/console sends {"enabled": false}; the explicit false value is always sent.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleModeSendsExplicitBooleanAndAcceptsSuccessCodes`
- 200, 202 and 204 succeed and other codes keep the native error; empty references fail before HTTP; cancellation makes no request.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleModeSendsExplicitBooleanAndAcceptsSuccessCodes`, `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleAndTraitRejectEmptyInputAndHonorCancellation`

Python/Go differences:

- Go exposes one SetConsoleMode(enabled) method for both Python entry points. Acknowledgement does not prove the console changed state; the caller reads GetConsole. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/drivers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.API.ListDrivers`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.WithListDriversOptions`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.WithListDriversQuery`

- GET /drivers sends detail=true only when requested (Python details=True maps to the detail query), the driver type and extension queries, and follows drivers_links next links.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverListFiltersFollowLinksAndExtensionQuery`
- A false Detail is omitted, nil options and empty query keys fail before HTTP; cancellation makes no request.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverListOmitsFalseDetailAndRejectsInvalidOptionsBeforeHTTP`, `api/ironic_drivers_conductors_test.go:TestIronicDriverCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python is a lazy generator of Driver Resources and rejects unknown query keys; Go is an iter.Seq2 of typed drivers and also accepts explicit extension queries. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically.

## `python:baremetal/v1/enable_node_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.SetConsoleMode`

- PUT /nodes/{id}/states/console sends {"enabled": true}; the explicit false value is always sent.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleModeSendsExplicitBooleanAndAcceptsSuccessCodes`
- 200, 202 and 204 succeed and other codes keep the native error; empty references fail before HTTP; cancellation makes no request.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleModeSendsExplicitBooleanAndAcceptsSuccessCodes`, `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleAndTraitRejectEmptyInputAndHonorCancellation`

Python/Go differences:

- Go exposes one SetConsoleMode(enabled) method for both Python entry points. Acknowledgement does not prove the console changed state; the caller reads GetConsole. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/find_chassis`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no chassis API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /chassis/{id} with a fallback list lookup, ignore_missing and details handling.

## `python:baremetal/v1/find_deploy_template`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no deploy template API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /deploy_templates/{id} with a fallback list lookup, ignore_missing and details handling.

## `python:baremetal/v1/find_port`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.ListDetail`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python find_port (deprecated) calls Resource.find with details=True: a GET by ID falls back to a detailed list filtered by the name argument and honors ignore_missing, duplicates and the details flag; the Collection Find lookup rules for ports have no test comparing that fallback sequence, query and duplicate handling.
- Python emits a RemovedInSDK50Warning for the method; Go has no corresponding signal.

## `python:baremetal/v1/find_port_group`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no port group API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /portgroups/{id} with a fallback list lookup, ignore_missing and details handling.

## `python:baremetal/v1/find_volume_connector`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume connector API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /volume/connectors/{id} with a fallback list lookup, ignore_missing and details handling.

## `python:baremetal/v1/find_volume_target`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no volume target API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /volume/targets/{id} with a fallback list lookup, ignore_missing and details handling.

## `python:baremetal/v1/get_allocation`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations.API.Get`

- GET /allocations/{id} decodes the typed model, accepts only 200 and keeps a native 404.
  Tests: `api/ironic_allocations_test.go:TestIronicAllocationGetAndDeleteStatusPolicies`, `api/ironic_allocations_test.go:TestIronicAllocationCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python get_allocation(allocation, fields=[...]) sends a comma-separated fields query; the Go allocations Get has no fields option, so field-limited reads are unmapped and untested.
- Python caps the microversion at 1.60; Go sends only the configured header.

## `python:baremetal/v1/get_chassis`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no chassis API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /chassis/{id} (with an optional fields query).

## `python:baremetal/v1/get_conductor`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors.API.Get`

- GET /conductors/{hostname} keeps alive=false, accepts only 200 and keeps a native 404.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicConductorGetKeepsFalseAndStatusPolicy`, `api/ironic_drivers_conductors_test.go:TestIronicConductorCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python get_conductor(conductor, fields=[...]) sends a comma-separated fields query; the Go conductors Get has no fields option, so field-limited reads are unmapped and untested.
- Python caps the microversion at 1.49; Go sends only the configured header.

## `python:baremetal/v1/get_deploy_template`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no deploy template API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /deploy_templates/{id} (with an optional fields query).

## `python:baremetal/v1/get_driver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers.API.GetDriverDetails`

- GET /drivers/{name} decodes the driver with its default interfaces; only 200 succeeds and a native 404 is kept.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverReadsDetailsPropertiesDiskPropertiesAndPassthruMethods`, `api/ironic_drivers_conductors_test.go:TestIronicDriverReadsRequireStatus200AndKeepNative404`
- Pre-canceled calls make no request.
  Tests: `api/ironic_drivers_conductors_test.go:TestIronicDriverCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python raises NotFoundException on 404; Go keeps the native unexpected-status error (the Collection Find adds ErrNotFound). Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry.

## `python:baremetal/v1/get_inspection_rule`

Status: `unresolved`.

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- The pinned Gophercloud v2.15.0 baremetal inventory has no inspection rule API and this module has no Go package for it, so there is no Go counterpart to compare; a package with concrete options, tests and documentation is needed.
- Python behavior to map: GET /inspection_rules/{id} (with an optional fields query).

## `python:baremetal/v1/get_node_boot_device`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.GetBootDevice`

- GET /nodes/{id}/management/boot_device decodes boot_device and persistent (false kept); the client microversion header is sent.
  Tests: `api/ironic_nodes_states_test.go:TestIronicNodeBootDeviceKeepsFalsePersistentAndDecodesReads`
- Only 200 succeeds; other codes keep the native unexpected-status error.
  Tests: `api/ironic_nodes_states_test.go:TestIronicNodeSuccessCodesAreExactForStateAndManagementCalls`
- Pre-canceled and expired contexts make no request and keep their cause.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python returns the raw response dict; Go returns BootDeviceOpts with the two documented fields. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry.

## `python:baremetal/v1/get_node_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.GetConsole`

- GET /nodes/{id}/states/console returns console_enabled (false kept) with console_info and the full body retained verbatim; no body or query is sent.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleReadKeepsInfoAndRequires200`
- Only 200 succeeds; an empty node reference fails before HTTP; pre-canceled contexts make no request.
  Tests: `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleReadKeepsInfoAndRequires200`, `api/py_ironic_a_nodes_test.go:TestPyIronicAConsoleAndTraitRejectEmptyInputAndHonorCancellation`

Python/Go differences:

- Python returns the raw dict; Go returns ConsoleState with Enabled, raw Info and the raw Body. Node identifiers are path-escaped by Go. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry. Python treats every status below 400 as success; Go accepts exactly the success codes listed in the contracts.

## `python:baremetal/v1/get_node_inventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.GetInventory`

- GET /nodes/{id}/inventory decodes inventory and plugin_data into the typed model.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeInventoryAndFirmwareDecodeNullableFields`
- Only 200 succeeds; other codes keep the native unexpected-status error.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites`
- Pre-canceled and expired contexts make no request and keep their cause.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python returns the response document as an untyped dict including every server key; Go decodes the native InventoryData fields only. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry.

## `python:baremetal/v1/get_node_supported_boot_devices`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.GetSupportedBootDevices`

- GET /nodes/{id}/management/boot_device/supported returns the supported_boot_devices list.
  Tests: `api/ironic_nodes_states_test.go:TestIronicNodeBootDeviceKeepsFalsePersistentAndDecodesReads`
- Only 200 succeeds; other codes keep the native unexpected-status error.
  Tests: `api/ironic_nodes_states_test.go:TestIronicNodeSuccessCodesAreExactForStateAndManagementCalls`
- Pre-canceled and expired contexts make no request and keep their cause.
  Tests: `api/ironic_nodes_related_test.go:TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause`

Python/Go differences:

- Python returns the raw dict {"supported_boot_devices": [...]}; Go returns the unwrapped string slice. Python selects a microversion capped by the resource class maximum; the Go service client sends only the configured Microversion header and never raises it automatically. Python retries HTTP 409/503 responses automatically (keystoneauth retriable status codes); Go never replays a failed or uncertain call, so the caller decides whether to retry.

## `python:baremetal/v1/get_port`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports.API.Get`

- GET /ports/{id} decodes the typed model, accepts only 200 and keeps a native 404; cancellation makes no request.
  Tests: `api/ironic_ports_test.go:TestIronicPortGetAndDeleteStatusPolicies`, `api/ironic_ports_test.go:TestIronicPortCancellationPreservesCauseAndSkipsHTTP`

Python/Go differences:

- Python returns Resource objects or raw dicts and responses; Go returns concrete typed models and errors, with native error causes kept and context cancellation observable.

Remaining:

- Python get_port(port, fields=[...]) maps field names to wire names and sends a comma-separated fields query; the Go ports Get takes no fields option, so field-limited reads are unmapped and untested.
- Python caps the microversion at 1.101; Go sends only the configured header.
