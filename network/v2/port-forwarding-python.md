# Port Forwarding Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_floating_ip_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordMutationOpts`

- Create posts exactly the supplied Body attrs in port_forwarding, preserving omitted/null/false/zero/empty values and unknown extensions without native required-field defaults; parent URI remains separate, Source descriptor view overlays the actual response on input while keeping Wire and envelope/status evidence.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingMutationsOwnAttributesDescriptorsAndParent`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/create_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordMutationOpts`

- Create posts exactly the supplied Body attrs in port_forwarding, preserving omitted/null/false/zero/empty values and unknown extensions without native required-field defaults; parent URI remains separate, Source descriptor view overlays the actual response on input while keeping Wire and envelope/status evidence.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingMutationsOwnAttributesDescriptorsAndParent`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/delete_floating_ip_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordDeleteIgnoreMissing`

- Delete sends one fixed scoped DELETE and defaults to suppressing only a clean 404; explicit ignore_missing=false retains it. Rejected-body read/close, retry, source changes and cancellation keep their original causes and native body/status proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingCleanOnlyFindAndDeleteRejectedBodyCauses`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/delete_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordDeleteIgnoreMissing`

- Delete sends one fixed scoped DELETE and defaults to suppressing only a clean 404; explicit ignore_missing=false retains it. Rejected-body read/close, retry, source changes and cancellation keep their original causes and native body/status proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingCleanOnlyFindAndDeleteRejectedBodyCauses`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/find_floating_ip_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindOpts`

- Find first GETs a literal member including the Source floatingip_id query and supplied kwargs, falling back only for a clean 400/403/404 to the exact-ID-or-name scoped list. Fallback applies declared server/local query classification and fully observes later errors and duplicates; missing defaults to nil and strict missing reports ErrNotFound.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingFindKeepsSourceFetchKwargsAndCompleteFallback`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingFindMissingDuplicateAndDirectSuccess`.
- Rejected response read/close, source mutation, retry callback or context causes remain errors with native response evidence and never trigger Find fallback.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingCleanOnlyFindAndDeleteRejectedBodyCauses`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.
- Dictionary links.next is an explicit Go compatibility extension; Source dictionary-link parsing only sees singleton key/value dictionaries without rel/href.

### `python:network/v2/find_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFindOpts`

- Find first GETs a literal member including the Source floatingip_id query and supplied kwargs, falling back only for a clean 400/403/404 to the exact-ID-or-name scoped list. Fallback applies declared server/local query classification and fully observes later errors and duplicates; missing defaults to nil and strict missing reports ErrNotFound.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingFindKeepsSourceFetchKwargsAndCompleteFallback`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingFindMissingDuplicateAndDirectSuccess`.
- Rejected response read/close, source mutation, retry callback or context causes remain errors with native response evidence and never trigger Find fallback.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingCleanOnlyFindAndDeleteRejectedBodyCauses`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.
- Dictionary links.next is an explicit Go compatibility extension; Source dictionary-link parsing only sees singleton key/value dictionaries without rel/href.

### `python:network/v2/floating_ip_port_forwardings`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListOpts`

- List is lazy and independently reiterable; declared query fields use Requests scalar/sequence encoding, other known Body kwargs filter projected rows locally, unknown kwargs are ignored, and fixed parent is present on every record. Raw max-items caps precede local filtering and a yield break or single-page option prevents reading continuation.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListProjectsQueriesFiltersAndLazyPagination`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListCapsBeforeFilterAndRejectsScopeEscape`.
- Source plural/body/HTTP links and marker fallback keep one fixed collection and inherited filters; malformed or cross-scope continuation failures keep whole-page proof, while reaching a cap does not inspect unconsumed continuations.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListProjectsQueriesFiltersAndLazyPagination`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListCapsBeforeFilterAndRejectsScopeEscape`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.
- Dictionary links.next is an explicit Go compatibility extension; Source dictionary-link parsing only sees singleton key/value dictionaries without rel/href.

### `python:network/v2/get_floating_ip_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFields`

- Get sends the exact escaped parent/member path with ordered repeated fields, preserving unknown field spellings and omitting an empty fields query. HTTP 200..399 follows Source response projection; raw integer fields retain original JSON while int descriptor views follow Source conversion and failures keep response proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/get_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordFields`

- Get sends the exact escaped parent/member path with ordered repeated fields, preserving unknown field spellings and omitting an empty fields query. HTTP 200..399 follows Source response projection; raw integer fields retain original JSON while int descriptor views follow Source conversion and failures keep response proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/port_forwardings`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordListOpts`

- List is lazy and independently reiterable; declared query fields use Requests scalar/sequence encoding, other known Body kwargs filter projected rows locally, unknown kwargs are ignored, and fixed parent is present on every record. Raw max-items caps precede local filtering and a yield break or single-page option prevents reading continuation.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListProjectsQueriesFiltersAndLazyPagination`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListCapsBeforeFilterAndRejectsScopeEscape`.
- Source plural/body/HTTP links and marker fallback keep one fixed collection and inherited filters; malformed or cross-scope continuation failures keep whole-page proof, while reaching a cap does not inspect unconsumed continuations.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListProjectsQueriesFiltersAndLazyPagination`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingListCapsBeforeFilterAndRejectsScopeEscape`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.
- Dictionary links.next is an explicit Go compatibility extension; Source dictionary-link parsing only sees singleton key/value dictionaries without rel/href.

### `python:network/v2/update_floating_ip_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordMutationOpts`

- Update sends only supplied Body attrs in a scoped PUT; empty attrs are a zero-HTTP ID-only no-op. PortForwarding inherits Resource rather than NetworkResource, so if_match and if_revision remain ordinary unknown Body attrs. Accepted bodyless acknowledgements keep known inputs without fabricating Wire.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingMutationsOwnAttributesDescriptorsAndParent`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

### `python:network/v2/update_port_forwarding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithRecordMutationOpts`

- Update sends only supplied Body attrs in a scoped PUT; empty attrs are a zero-HTTP ID-only no-op. PortForwarding inherits Resource rather than NetworkResource, so if_match and if_revision remain ordinary unknown Body attrs. Accepted bodyless acknowledgements keep known inputs without fabricating Wire.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingMutationsOwnAttributesDescriptorsAndParent`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors`.
- Literal parent/member IDs stay on the selected Neutron scope, parameters are copied before HTTP, invalid Resource/route controls and canceled or changed selected sources fail without dispatch, and accepted handling failures retain actual HTTP proof.
  Tests: `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation`, `network/v2/extensions/layer3/portforwarding/python_records_contracts_test.go:TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy`.

Python/Go differences:

- Go fixes the floating IP as an explicit literal parent ID in every signature and returns an owned PortForwardingRecord instead of accepting or mutating Python Resource objects; inherited mutable Resource/session parity remains the shared R1 work item.
- The Go descriptor view keeps location null instead of injecting connection-derived project metadata; ParentID stores the URI component separately. Unknown Body extensions and original HTTP evidence remain available.
- Go escapes ID path segments and rejects blank/control/dot IDs and mutable route/scope/transport controls before HTTP; pagination cannot change collection path, parent, filters or origin. Find base_path/list_base_path/microversion kwargs are rejected rather than replacing the fixed service route.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
created = conn.network.create_floating_ip_port_forwarding(
    floating_ip_id, internal_port_id=port_id,
    internal_ip_address="10.0.0.5", internal_port=22,
    external_port=2222, protocol="tcp", description="SSH")
selected = conn.network.get_floating_ip_port_forwarding(
    floating_ip_id, created.id, fields=["id", "external_port"])
for forwarding in conn.network.floating_ip_port_forwardings(
    floating_ip_id, protocol="tcp", description="SSH"):
    print(forwarding.id)
found = conn.network.find_floating_ip_port_forwarding(floating_ip_id, created.id)
#     floating_ip_id, created.id, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
)

func printForwarding(value *portforwarding.PortForwardingRecord) {
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("parent=%s status=%d\n%s\n", value.ParentID, value.StatusCode, body)
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    floatingIP := flag.String("floating-ip", "", "floating IP ID")
    id := flag.String("id", "", "port forwarding ID to get and find")
    port := flag.String("port", "", "internal Neutron port ID")
    address := flag.String("address", "10.0.0.5", "internal IP address")
    create := flag.Bool("create", false, "create a port forwarding")
    description := flag.String("description", "", "replace description, including empty")
    remove := flag.String("delete", "", "port forwarding ID to delete")
    flag.Parse()
    if *floatingIP == "" { log.Fatal("-floating-ip is required") }
    update := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "description" { update = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.PortForwarding
    for row, err := range api.ListRecords(ctx, *floatingIP,
        portforwarding.WithRecordListQuery("protocol", "tcp"),
        portforwarding.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *id != "" {
        row, err := api.GetRecord(ctx, *floatingIP, *id,
            portforwarding.WithRecordFields("id", "external_port"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
        found, err := api.FindRecord(ctx, *floatingIP, *id)
        if err != nil { log.Fatal(err) }
        printForwarding(found)
        if update {
            row, err := api.UpdateRecord(ctx, *floatingIP, *id,
                portforwarding.WithRecordAttribute("description", *description))
            if err != nil { log.Fatal(err) }
            printForwarding(row)
        }
    }
    if *create {
        row, err := api.CreateRecord(ctx, *floatingIP,
            portforwarding.WithRecordAttribute("internal_port_id", *port),
            portforwarding.WithRecordAttribute("internal_ip_address", *address),
            portforwarding.WithRecordAttribute("internal_port", 22),
            portforwarding.WithRecordAttribute("external_port", 2222),
            portforwarding.WithRecordAttribute("protocol", "tcp"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *floatingIP, *remove,
            portforwarding.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Reference](README.md)
- [native CRUD](native-crud.md)
