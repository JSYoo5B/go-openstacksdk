# Python Parity Catalog

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/associate_endpoint_with_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.Create`

- Create sends a bodyless PUT OS-EP-FILTER/projects/{project}/endpoints/{endpoint} like Python Project.associate_endpoint, succeeds on 204 and returns 404 and 409 as errors.
  Tests: `identity/v3/projectendpoints/python_parity_test.go:TestPythonProjectEndpointAssociation`.

Python/Go differences:

- Python accepts any status below 400; the native call accepts only 204, so 200 or 202 becomes an error in Go.
- Python takes project and endpoint as ID strings or Resources; Go takes ID strings and returns only an error.

### `python:identity/v3/create_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.CreateConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- CreateConfigRecord PUTs the selected Domain singleton with config envelope, raw identity/ldap payloads including unknown nested settings and nullable values, then overlays the actual response and projects declared child Resource views without fabricating physical fields.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigCreateUsesSingletonPUTAndRawChildPayload`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetProjectsChildrenAndPreservesActualEvidence`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.

### `python:identity/v3/create_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithCreateField`

- Create sends POST endpoints with only the given interface/enabled/service_id/url keys plus region_id through WithCreateField in the {"endpoint": {...}} envelope, the request Python create_endpoint sends, and decodes id, interface, region_id, service_id, url and enabled.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Go CreateOpts.Region sends the legacy region key; the region_id key Python sends is added with endpoints.WithCreateField("region_id", id).
- Go requires interface, url and service_id before HTTP, while Python sends whatever attributes were given and lets Keystone reject the request.
- Python drops attributes Endpoint does not declare, such as description; Go CreateOpts can send description.

### `python:identity/v3/create_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.BatchCreate`

- A one-row BatchCreate sends POST limits with {"limits": [{...}]}, the body Python create_limit builds by wrapping its single limit, and element 0 of the result carries the fields Python reads from the first response row.
  Tests: `identity/v3/limits/python_parity_test.go:TestPythonLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Go returns the whole decoded slice; Python keeps only the first response row.
- Go requires service_id and resource_name before HTTP and always sends resource_limit, even 0, while Python sends only the given attributes.
- Go extension fields from WithBatchCreateField land beside the limits array, not inside the row.

### `python:identity/v3/create_region`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Create`

- Create sends POST regions with id, description and parent_region_id in the {"region": {...}} envelope, the request Python create_region sends, and decodes the fields Python exposes.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Go CreateOpts omits empty values, so Python attributes explicitly set to None or empty strings are not sent.

### `python:identity/v3/create_registered_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.BatchCreate`

- A one-row BatchCreate sends POST registered_limits with {"registered_limits": [{...}]}, the body Python create_registered_limit builds by wrapping its single registered limit, and element 0 of the result carries the fields Python reads from the first response row.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Go returns the whole decoded slice; Python keeps only the first response row.
- Go requires service_id and resource_name before HTTP and always sends default_limit, even 0, while Python sends only the given attributes.
- Go extension fields from WithBatchCreateField land beside the registered_limits array, not inside the row.

### `python:identity/v3/create_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Create`

- Create sends POST services with name, type, description and enabled in the {"service": {...}} envelope, the request Python create_service sends, and decodes the fields Python exposes.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Go CreateOpts.Type has no omitempty, so Go always sends type (as "" when empty) while Python omits an unset type.
- Without an extra object in the response, Go Service.Extra also copies name and description.

### `python:identity/v3/delete_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.DeleteConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- DeleteConfigRecord performs bodyless singleton DELETE, defaults to ignoring only clean404 and offers strict missing behavior. Accepted deletion and ignored404 retain independent actual response receipt; Python None becomes a Go Deleted/Response carrier.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigDeleteMissingDefaultAndStrictPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigDeleteAcceptedResponseIsIndependentReceipt`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.
- Python delete returns None; Go Deleted is true only on observed accepted deletion and false on ignored404. Deleted=false with an error does not prove nonexecution. Physical read/Close/cancel/source failures are never suppressed as logical absence.

### `python:identity/v3/delete_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE endpoints/{id}, succeeds on 204 and on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and returns a 409 error unchanged.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204. Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE limits/{id}, succeeds on 204 and on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and returns a 409 error unchanged.
  Tests: `identity/v3/limits/python_parity_test.go:TestPythonLimitDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204. Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_region`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE regions/{id}, succeeds on 204 and on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and returns a 409 error unchanged.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204. Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_registered_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE registered_limits/{id}, succeeds on 204 and on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and returns a 409 error unchanged.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204. Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE services/{id}, succeeds on 204 and on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and returns a 409 error unchanged.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204. Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/disassociate_endpoint_from_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.Delete`

- Delete sends a bodyless DELETE OS-EP-FILTER/projects/{project}/endpoints/{endpoint} like Python Project.disassociate_endpoint, succeeds on 204 and returns 404 and 409 as errors because Python has no ignore_missing here.
  Tests: `identity/v3/projectendpoints/python_parity_test.go:TestPythonProjectEndpointAssociation`.

Python/Go differences:

- Python accepts any status below 400; the native call accepts only 204, so 200 or 202 becomes an error in Go.
- Python takes project and endpoint as ID strings or Resources; Go takes ID strings and returns only an error.

### `python:identity/v3/endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.ProxyListRecords`

- List with ListOpts sends GET endpoints?interface=&region_id=&service_id= like Python endpoints(interface=..., region_id=..., service_id=...), follows links.next, and the endpoints collection maps those query keys while rejecting limit and marker before HTTP.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointListQueryAndPaging`.
- Concrete Source Endpoint MRO owns eight Body descriptors, null defaults, bool enabled alias order, untyped links, unknown discard, separate Wire/page/header/status; query limit/marker/interface/region_id/service_id vs declared local Body filters. Python-loaded equality and raw cap precede local filtering. Source dict links ignored, root/plural/top/HTTP link precedence and limit-driven marker continue to empty page. Lazy opaque generator JMESPath preserves identity and zero HTTP until consumption. SDK-owned callback/header/microversion/route snapshots and source/context guards retain actual physical read/Close/rejection evidence without hiding faults in native auth/retry/backoff; clean retries and explicit Source status acceptance preserve old native API.
  Tests: `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceMRODefaultsUntypedLinksAndActualEvidence`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOrderedEnabledAliasesAndPythonBool`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsQueriesLocalBodyFiltersAndUnknownKwargs`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceDictionaryLinksAndMarkerFallback`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceLinkPrecedenceAndMalformedPageReceipt`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsRawCapLazyBreakAndCancellation`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsMalformedAndLaterErrorDiscardPartialCollection`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOpaqueProxyJMESUsesActualGenerator`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOwnOptionsFormalControlsAndIdentityVersion`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceGuardBeforeOptionsAndAfterGeneratorCapture`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsPhysicalFaultsRetainActualProofWithoutHooks`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsCleanNativeRetryKeepsScopeQueryAndHeaders`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceStatusOptInPreservesNativeBindings`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Go concrete Source record splits defaults/projection from actual Wire/page/header/status; direct ServiceClient has SDK-owned location null. Optional HTTP cache and mutable Python Resource/Connection lifecycle remain shared SDK-R1/C1; no actual cloud acceptance claim.
- Go safe owned route/origin/cursor/auth guards reject Source arbitrary session/origin movement and invalid marker with actual page receipt. Generic options expose Source formal kwargs but do not permit receiver/session/auth replacement; Go JSON cannot construct arbitrary Python object operands.
- Opaque generator and serializable JSON are separate result carriers. Existing native API remains separate and unchanged; this named list is not full SDK parity.
- Verified Keystone27.0.0 list_endpoints is admin_required or role:reader and system_scope:all with system/project scope declarations. This is a system-reader user read, not general project-reader permission; deployments may override policy.

### `python:identity/v3/find_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) followed on nil by Find(resource.Name(x)) sends GET endpoints/{x} and then GET endpoints without query (Endpoint maps no name query), returns the ID or unique name match, nil when missing with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointFindIDThenNameFallback`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the list fallback in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller must continue to the name lookup only after a nil result.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- Go follows Keystone links.next during the name lookup; Python's list fallback does not.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

### `python:identity/v3/find_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) followed on nil by Find(resource.Name(x)) sends GET services/{x} and then GET services?name=x like Python (Service maps name as a query parameter), returns the ID or unique name match, nil when missing with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceFindIDThenNameFallback`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the list fallback in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller must continue to the name lookup only after a nil result.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- Go follows Keystone links.next during the name lookup; Python's list fallback does not.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

### `python:identity/v3/get_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.GetConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- GetConfigRecord GETs the singleton without an extra resource ID, preserves absent/null/passive id and actual envelope/header/status, and projects declared child mapping/defaults independently from raw Wire. Null, nonmapping scalar and empty-object child values remain distinct.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetProjectsChildrenAndPreservesActualEvidence`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetNullScalarAndEmptyChildrenHaveDistinctViews`.
- Malformed nested child projection after an accepted response retains actual ResponseError proof; strict GET404 is not ignored or converted into fallback/missing success.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigMalformedNestedChildKeepsAcceptedResponse`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.

### `python:identity/v3/get_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Get`

- Get sends GET endpoints/{id}, the request Python get_endpoint sends, and decodes the fields Python exposes.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.

### `python:identity/v3/get_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Get`

- Get sends GET limits/{id}, the request Python get_limit sends, and decodes the fields Python exposes.
  Tests: `identity/v3/limits/python_parity_test.go:TestPythonLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.

### `python:identity/v3/get_region`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Get`

- Get sends GET regions/{id}, the request Python get_region sends, and decodes the fields Python exposes.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.

### `python:identity/v3/get_registered_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Get`

- Get sends GET registered_limits/{id}, the request Python get_registered_limit sends, and decodes the fields Python exposes.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.

### `python:identity/v3/get_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Get`

- Get sends GET services/{id}, the request Python get_service sends, and decodes the fields Python exposes.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.

### `python:identity/v3/limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithRecordListPaginated`

- All with resource.WithQuery forwards service_id, project_id, domain_id and limit to GET limits as Python limits(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/limits/python_parity_test.go:TestPythonLimitListQueryAndPaging`.
- Pinned Proxy list performs lazy GET on fixed Keystone v3 collection with no implicit limit/discovery; only declared service/region/resource-name/project QueryParameters reach server. Unknown kwargs discard before marshaler. Source declarations are all untyped including documented numeric resource_limit; missing null, arbitrary JSON primitive and literal Wire/page/Header/status evidence are independent.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsLazyQueryAndUntypedView`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Source ignores root Keystone links:{next:url} dictionaries, honors root links key precedence over plural_links, follows short nonempty pages with explicit positive limit using last raw wire ID even when locally filtered, and stops at empty page. Default paginated true/limit omission/max_items zero, single-page and raw prefilter max-items limit hint retain Source behavior; unsafe next URL yields actual last page proof.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsDictionaryLinksIgnoredWithSourcePrecedence`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsFilteredMarkerAndRawMaxItems`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`.
- Only declared unmapped Body kwargs filter row views locally, never server query. Ordered recursive dictionary subset filters preserve falsey/missing/empty-dictionary/scalar errors and short circuit; scalar/nested comparisons use CPython-loaded JSON float and bool equality without changing raw evidence.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsLoadedLocalFilterAndShortCircuit`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsFilteredMarkerAndRawMaxItems`.
- Copied functional opts preserve query raw bytes, maps and pointers; Source headers/microversion/base_path formals and one-pass __conflicting_attrs restoration normalize through shared ABI. Unknown Source control-like names are discarded without encoding, bound arg collisions and restored duplicate paginated/base_path fail before HTTP. Explicit version controls identity header on all pages with no invented discovery.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsOptionsSnapshotFormalRestorationAndHeaders`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`.
- ProxyListRecords evaluates truthy string JMESPath against an opaque unconsumed Source generator before HTTP: projection/index/type return null, constants return JSON, length/to_string/sort_by fail, and @/to_array/not_null preserve generator identity. Omitted/falsey/nonstring filters keep lazy generator. JSON remains absent for generators; typed ListRecords explicitly directs truthy strings to polymorphic method.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitProxyListOpaqueProjectionAndRetainedGenerators`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Standalone no-Connection location defaults null and server location stays only in Wire. Immutable explicit CloudLocation uses existing ForResource with declared resource project_id, keeping falsey current scope and clearing names for foreign scope. Invalid raw location fails before HTTP; there is no auth/Connection metadata inference.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsLocationIsExplicitImmutableAndDeclared`.
- Singleton plural object accepted; missing/null/scalar envelopes and cls/connection/microversion/_synchronized row collisions retain actual accepted status/body/header evidence. AllRecords discards partial rows; iterators retain earlier yielded rows. Nil callbacks/context, selected source mutation and cancellation stop before subsequent option/HTTP; Close cancellation preserves caller cause and actual physical receipt.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsSingletonMalformedAndConstructorEvidence`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsCloseCancellationKeepsReceiptAndCause`.
- Actual 200/203/401/404/429/503 Read/Close/mixed physical faults retain every cause plus accepted receipt/native rejection body/header/status, with one Close/HTTP and zero Retry/Reauth callbacks. Native401 pointer preserves ErrOriginal/ErrReauth. Clean401 renewal and503 retry preserve fixed query/Accept/condition/version and original mutable service configuration.
  Tests: `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`, `identity/v3/limits/python_records_test.go:TestPythonLimitListRecordsCleanReauthenticationAndRetryKeepPolicy`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Python filters domain_id locally because Limit does not map it as a query parameter; Go WithQuery sends domain_id to Keystone as a server filter.
- Source returns mutable Resource instances and polymorphic generator results. Go preserves typed native List/All separately and exposes owned ListRecords/AllRecords + ProxyListRecords with complete Source JSON view and original evidence; truthy JMESPath uses the explicit polymorphic method. Go iter.Seq2 can be iterated again to repeat the captured operation; Python generator consumption is one-shot.
- A standalone native client has no Python Connection: default computed location is null, and WithRecordListLocation supplies immutable recorded facts as an explicit Go extension. No automatic cloud/project name inference, token lookup, auth scope change or SDK-R1/C1 closure is claimed.
- Go validates functional options and JSON transport values before HTTP, requires safe fixed-origin service-relative paths, finite representable identifiers and nonnegative integer pagination amounts. Unsafe URLs/constructor collisions/malformed JSON become explicit receipt-bearing Go errors rather than Python exception classes or odd negative/noninteger pagination behavior.
- This class has no maximum microversion/discovery profile. Go honors selected native default and typed per-call microversion/header options while guarding selected source and SDK-owned auth/route/protocol headers. Dynamic Connection/session/Resource dirty/cache/warning contracts remain tracked at whole SDK level.

### `python:identity/v3/regions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.RegionRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListJMESPath`

- All with resource.WithQuery forwards parent_region_id, limit and marker to GET regions as Python regions(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionListQueryAndPaging`.
- Region inherits Resource directly with raw nullable description/links/parent_region_id/id/name and no declared Headers. Parent region is server query; name is local Body, unknown kwargs are discarded before marshal, and independent declared/Wire/Envelope/header/status channels retain actual proof.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsRawDefaultsLocalNameAndIndependentEvidence`.
- Source root dictionary links ignored with root-key precedence, null is an iterable error with proof, explicit short limit pages continue by last raw ID until empty and local-filtered last row still supplies cursor; raw max_items and break stop lookahead.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsSourceDictionaryLinksAndPresence`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsFilteredMarkerRawCapAndBreak`.
- Loaded ordered local subset/equality follows CPython float rounding/underflow and bool/integer equality; earlier mismatch avoids later bad shape and reached shape failure has actual proof.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsOrderedLoadedLocalEquality`.
- Opaque unconsumed generator JMESPath preserves HTTP0 projection/type/literal errors/results and generator/nested identity without fake JSON or receipt; falsey/nonstring remains lazy.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Typed/formal controls and one-pass conflicting attrs preserve restored parent query/header/version, original expression priority, unknown discard and HTTP0 collisions. Options/source/cancel before and after callbacks are sticky with owned maps/raw JSON.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsProxyFormalRestorationAndBoundCollision`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsOwnedOptionsStickySourceAndCancellation`.
- Explicit location clones configured current project, ignores unknown wire project/zone and normalizes no-descriptor zone null. New profile opts Source valid100..399; raw bool limit is intentionally decimal-Go-only HTTP0 invalid rather than wire1/0.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsExplicitLocationStatusAndBooleanLimitBoundary`.
- 28 physical accepted/rejected Read/Close/restored Source/custom cancel lanes keep actual proof, Close once, native401 independently inspectable ErrOriginal/ErrReauth and zero retry/backoff/reauth. Clean401/503/429/498 retains live token and owned route/header/query.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsPhysicalFaultsKeepActualProofAndStopRetry`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Library-owned concrete functional options, raw nullable Source view and separate actual receipt preserve existing native APIs. No user builder interface or new transport framework.
- Go caps/limit require nonnegative decimal integers. Source raw limit True/False is sent as True/False and truthy True activates fallback; Go rejects bool before HTTP and does not substitute1/0. Safe fixed origin/path/query/header/cursor/cycle bounds are explicit.
- Source rootlink JSON precedence/dictionary-ignore/null-error is preserved. RFC HTTP Link uses explicit Go interoperability for ordinary Requests next[url] versus Source next[uri] bug.
- Unconsumed Proxy generator is opaque and no fake JSON/HTTP association is created. Go iter.Seq2 reuse reissues captured requests instead of sharing Python one-shot state.
- Standalone default location is null. Explicit cloned recorded facts are a Go convenience; Region has no zone descriptor and normalizes zone null, while GroupUser reuses existing User explicit-zone convenience. Unknown response scope fields are not trusted.
- Already selected Keystone v3 native API required. Mutable Resource object/group subclass routing, automatic Connection location/cache, generic synchronized lifecycle and session remain SDK-R1/C1/S1.
- Generic SDK-R1/C1/S1 response cache, arbitrary synchronized Resource/object lifecycle and automatic Connection/session bridge are not completed. These two finite mappings have supported delta0; root Go/race/main gates must pass before catalog promotion.

### `python:identity/v3/registered_limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListPaginated`

- All with resource.WithQuery forwards service_id, resource_name and marker to GET registered_limits as Python registered_limits(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitListQueryAndPaging`.
- Pinned Proxy list performs lazy GET on fixed Keystone v3 collection with no implicit limit/discovery; only declared service/region/resource-name QueryParameters reach server. Unknown kwargs discard before marshaler. Source declarations are all untyped including documented numeric default_limit; missing null, arbitrary JSON primitive and literal Wire/page/Header/status evidence are independent.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLazyQueryAndUntypedView`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Source ignores root Keystone links:{next:url} dictionaries, honors root links key precedence over plural_links, follows short nonempty pages with explicit positive limit using last raw wire ID even when locally filtered, and stops at empty page. Default paginated true/limit omission/max_items zero, single-page and raw prefilter max-items limit hint retain Source behavior; unsafe next URL yields actual last page proof.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsDictionaryLinksIgnoredWithSourcePrecedence`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsFilteredMarkerAndRawMaxItems`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`.
- Only declared unmapped Body kwargs filter row views locally, never server query. Ordered recursive dictionary subset filters preserve falsey/missing/empty-dictionary/scalar errors and short circuit; scalar/nested comparisons use CPython-loaded JSON float and bool equality without changing raw evidence.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLoadedLocalFilterAndShortCircuit`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsFilteredMarkerAndRawMaxItems`.
- Copied functional opts preserve query raw bytes, maps and pointers; Source headers/microversion/base_path formals and one-pass __conflicting_attrs restoration normalize through shared ABI. Unknown Source control-like names are discarded without encoding, bound arg collisions and restored duplicate paginated/base_path fail before HTTP. Explicit version controls identity header on all pages with no invented discovery.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsOptionsSnapshotFormalRestorationAndHeaders`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`.
- ProxyListRecords evaluates truthy string JMESPath against an opaque unconsumed Source generator before HTTP: projection/index/type return null, constants return JSON, length/to_string/sort_by fail, and @/to_array/not_null preserve generator identity. Omitted/falsey/nonstring filters keep lazy generator. JSON remains absent for generators; typed ListRecords explicitly directs truthy strings to polymorphic method.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitProxyListOpaqueProjectionAndRetainedGenerators`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Standalone no-Connection location defaults null and server location stays only in Wire. Immutable explicit CloudLocation uses existing ForResource; unknown project_id cannot override configured current scope. Invalid raw location fails before HTTP; there is no auth/Connection metadata inference.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLocationIsExplicitImmutableAndDeclared`.
- Singleton plural object accepted; missing/null/scalar envelopes and cls/connection/microversion/_synchronized row collisions retain actual accepted status/body/header evidence. AllRecords discards partial rows; iterators retain earlier yielded rows. Nil callbacks/context, selected source mutation and cancellation stop before subsequent option/HTTP; Close cancellation preserves caller cause and actual physical receipt.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSingletonMalformedAndConstructorEvidence`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsCloseCancellationKeepsReceiptAndCause`.
- Actual 200/203/401/404/429/503 Read/Close/mixed physical faults retain every cause plus accepted receipt/native rejection body/header/status, with one Close/HTTP and zero Retry/Reauth callbacks. Native401 pointer preserves ErrOriginal/ErrReauth. Clean401 renewal and503 retry preserve fixed query/Accept/condition/version and original mutable service configuration.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsCleanReauthenticationAndRetryKeepPolicy`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Source returns mutable Resource instances and polymorphic generator results. Go preserves typed native List/All separately and exposes owned ListRecords/AllRecords + ProxyListRecords with complete Source JSON view and original evidence; truthy JMESPath uses the explicit polymorphic method. Go iter.Seq2 can be iterated again to repeat the captured operation; Python generator consumption is one-shot.
- A standalone native client has no Python Connection: default computed location is null, and WithRecordListLocation supplies immutable recorded facts as an explicit Go extension. No automatic cloud/project name inference, token lookup, auth scope change or SDK-R1/C1 closure is claimed.
- Go validates functional options and JSON transport values before HTTP, requires safe fixed-origin service-relative paths, finite representable identifiers and nonnegative integer pagination amounts. Unsafe URLs/constructor collisions/malformed JSON become explicit receipt-bearing Go errors rather than Python exception classes or odd negative/noninteger pagination behavior.
- This class has no maximum microversion/discovery profile. Go honors selected native default and typed per-call microversion/header options while guarding selected source and SDK-owned auth/route/protocol headers. Dynamic Connection/session/Resource dirty/cache/warning contracts remain tracked at whole SDK level.

### `python:identity/v3/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.ServiceRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.RecordListGenerator.JMESPathIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.RecordListGenerator.MarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ProxyListRecords`

- All with resource.WithQuery forwards name, type and limit to GET services as Python services(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceListQueryAndPaging`.
- Pinned Service nullable id/name/description/type/links/is_enabled descriptors retain untyped JSON, bool string truthiness and response canonical/wire last-member precedence; unknown Body is absent from view but physical Wire/page/header/status/Clone stay independently owned.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsBoolAliasResponseOrderAndLocalCanonicalKey`.
- Only name/type and inherited limit/marker are server query. Canonical is_enabled and other declared Body values are ordered loaded local filters; undeclared enabled wire query and unknown kwargs discard before encoding. Recursive empty-dictionary/truthy-scalar and loaded float equality match Source.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsDeclaredQueryAndOrderedLoadedLocalFilters`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsBoolAliasResponseOrderAndLocalCanonicalKey`.
- Source dictionary root links ignored with key-presence precedence, positive limit marker from last raw wire ID even short pages, max_items raw prefilter cap, empty terminal, lazy break and reusable captured iterator work through shared pager.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsRawMarkerShortPagesCapBreakAndReusableIterator`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsSourceDictionaryRootPresenceAndActualContinuationGuards`.
- Immutable option snapshots/typed or raw formal headers/MV/base path and one-pass conflicting attrs match the Source kwargs control cycle. Singleton plural rows, missing plural/204/malformed/constructor collision and local filter failure retain actual accepted receipt; nil/source/cancel preflight errors have no fabricated receipt.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsOptionSnapshotsFormalsAndPreflightSourceGuards`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsFormalRestoreHeaderVersionUnknownAndConstructorReceipt`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage`.
- Proxy truthy string JMES sees opaque unconsumed generator before HTTP, preserving null projection/type/constants/identity/nested generators and explicit length type-error; omitted/falsey/nonstring keep lazy generator. Typed record iterator requires polymorphic counterpart when return type changes.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity`.
- Default standalone location is SourceNoConnection null. Immutable explicit current project facts ignore undeclared remote project/location; invalid JSON scope validates before HTTP, while literal Wire remains original.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordOptionsSnapshotCancellationAndExplicitLocation`.
- Physical read/close/mixed fault matrix preserves actual response and causes, native401 pointer ErrOriginal/ErrReauth and one Close/request, with zero Retry/Reauth/Backoff. Clean401/503/429/498 renewal/retry preserve captured query/header/MV and live token.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- iter.Seq2 can repeat a captured request on another iteration instead of a one-shot Python generator. Proxy result Value may retain opaque/nested generator identity and JSON is nil if nonserializable. Python opaque repr/to_string(@) is deliberately bounded by Go serialization errors; no independent Python JMES parser/runtime parity is asserted.

### `python:identity/v3/update_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.UpdateConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- UpdateConfigRecord PATCHes raw explicit identity/ldap values and null in a config envelope, overlays actual response on seed and discards dirty id from wire while retaining the explicit seed id view. No recognized dirty fields yields no HTTP and no invented receipt.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigUpdatePATCHNullAndSeedOverlay`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigUpdateEmptyAndUnknownAttributesAvoidHTTP`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.
- Update is a newly seeded singleton operation, not mutation/equality/recommit of an existing Python Resource. An optional attrs id may seed the view but is removed from commit body; the parent domain_id URI remains bound independently.

### `python:identity/v3/update_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithUpdateField`

- Update sends PATCH endpoints/{id} with only the changed keys in the {"endpoint": {...}} envelope, region_id through WithUpdateField, the request Python update_endpoint sends.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Python commit sends no request when no attribute changed and never puts id in the PATCH body; Go Update always sends PATCH, with an empty envelope for empty options.
- region_id is sent with endpoints.WithUpdateField("region_id", id) because UpdateOpts.Region sends the legacy region key.
- Go UpdateOpts omits empty strings and WithUpdateField cannot override typed keys, so Python None or empty values for interface, url or service_id cannot be sent.

### `python:identity/v3/update_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Update`

- Update sends PATCH limits/{id} with only the changed keys in the {"limit": {...}} envelope, including zero resource_limit and empty description, the request Python update_limit sends.
  Tests: `identity/v3/limits/python_parity_test.go:TestPythonLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Python commit sends no request when no attribute changed and never puts id in the PATCH body; Go Update always sends PATCH, with an empty envelope for empty options.
- Python can send any Limit attribute; Go UpdateOpts types only description and resource_limit, and other keys such as region_id go through WithUpdateField.

### `python:identity/v3/update_region`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Update`

- Update sends PATCH regions/{id} with only the changed keys in the {"region": {...}} envelope, including an empty description through the pointer field, the request Python update_region sends.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Python commit sends no request when no attribute changed and never puts id in the PATCH body; Go Update always sends PATCH, with an empty envelope for empty options.
- UpdateOpts.ParentRegionID omits empty values and WithUpdateField rejects parent_region_id, so Python update_region(parent_region_id=None) cannot be sent.

### `python:identity/v3/update_registered_limit`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Update`

- Update sends PATCH registered_limits/{id} with only the changed keys in the {"registered_limit": {...}} envelope, including zero default_limit, the request Python update_registered_limit sends.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Python commit sends no request when no attribute changed and never puts id in the PATCH body; Go Update always sends PATCH, with an empty envelope for empty options.
- UpdateOpts.RegionID, ServiceID and ResourceName omit empty values and WithUpdateField rejects those keys, so Python None values for them cannot be sent.

### `python:identity/v3/update_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Update`

- Update sends PATCH services/{id} with only the changed keys in the {"service": {...}} envelope, including empty description and false enabled, the request Python update_service sends.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v3/native-admin-b.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Keystone v3 catalog resources declare no microversion, so neither side negotiates or sends one.
- Python commit sends no request when no attribute changed and never puts id in the PATCH body; Go Update always sends PATCH, with an empty envelope for empty options.
- UpdateOpts.Type omits an empty value and WithUpdateField rejects type, so Python type=None cannot be sent.

### `python:identity/v3/wait_for_delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`

- Region WaitForDeletion with WithTimeout(120s) repeats GET regions/{id} on the fixed ID until 404, reports progress 0 after the nonterminal observation and then succeeds.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionWaitHelpers`.

Python/Go differences:

- Go defaults to a five-minute timeout while the identity v3 Proxy default is wait=120; resource.WithTimeout reproduces it.

Unresolved boundaries:

- Python accepts every identity v3 Resource; Go has separate per-package helpers and none for packages outside a Resources collection.
- Python returns the original Resource on deletion; Go returns only an error.

### `python:identity/v3/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`

- Endpoint and Region WaitFor without a status attribute fail with resource.ErrUnsupported before HTTP, like the Python AttributeError for a missing status attribute.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointWaitHelpers`, `identity/v3/regions/python_parity_test.go:TestPythonRegionWaitHelpers`.
- Region WaitFor with WithStatusAttribute("description") polls GET regions/{id} on the fixed ID and compares the selected field case-insensitively, like Python attribute selection with lowercased status values.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionWaitHelpers`.

Python/Go differences:

- Go defaults to a five-minute timeout; Python wait=None waits forever, which Go expresses with resource.WithUnlimitedWait. resource.WithPollInterval reproduces the 2-second interval.

Unresolved boundaries:

- Python accepts every identity v3 Resource; Go has separate per-package helpers and none for packages outside a Resources collection.
- Python returns the given Resource without HTTP when its cached attribute already equals the target; Go always sends the first GET and returns a model pointer instead of the refreshed Resource.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
// Python: conn.identity.delete_region("RegionOne")
err := regions.New(client).Remove(ctx, resource.ID("RegionOne"))
// Python: conn.identity.delete_region("RegionOne", ignore_missing=False)
err = regions.New(client).Remove(ctx, resource.ID("RegionOne"), resource.WithMissingError())
```

```go
disabled := false
ep, err := endpoints.New(client).Create(ctx,
	endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: "svc-1", Enabled: &disabled},
	endpoints.WithCreateField("region_id", "RegionOne"))
```

```go
rows, err := limits.New(client).BatchCreate(ctx, limits.BatchCreateOpts{
	{ServiceID: "svc-1", ProjectID: "p", ResourceName: "cores", ResourceLimit: 10},
})
limit := rows[0]
```

## References

- [identity v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [Native Admin B](native-admin-b.md)
- [Limit](limits/python_records.md)
- [RegisteredLimit](registeredlimits/python_records.md)
- [Endpoint](endpoints/python-records.md)
- [Service](services/python-records.md)
- [Region](regions/python-records.md)
- [`create_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L404)
- [`delete_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L415)
- [`find_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L448)
- [`get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L468)
- [`endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L483)
- [`update_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L497)
- [`endpoint_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1130)
- [`project_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L537)
- [`associate_endpoint_with_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L557)
- [`disassociate_endpoint_from_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L574)
- [`create_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1594)
- [`delete_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1605)
- [`find_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1636)
- [`get_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1660)
- [`regions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1672)
- [`update_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1683)
- [`create_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1232)
- [`delete_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1243)
- [`find_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274)
- [`get_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1293)
- [`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1305)
- [`update_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1319)
- [`create_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2523)
- [`delete_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2548)
- [`get_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2511)
- [`limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2501)
- [`update_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2534)
- [`create_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2442)
- [`delete_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2474)
- [`get_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2425)
- [`registered_limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2411)
- [`update_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2455)
- [`create_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L593)
- [`delete_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L606)
- [`find_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L645)
- [`get_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L668)
- [`endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L684)
- [`update_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L697)
- [`project_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L714)
- [`endpoint_group_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L511)
- [`endpoint_group_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1166)
- [`associate_project_with_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1190)
- [`disassociate_project_from_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1210)
- [`create_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L317)
- [`delete_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L339)
- [`get_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L362)
- [`update_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L382)
- [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3343)
- [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3382)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Python Records](endpoints/python-records.md)
- [Cloud Endpoints Python](../cloud-endpoints-python.md)
- [Python Records](regions/python-records.md)
- [Python Records](services/python-records.md)
- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
