# Endpointgroups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/associate_project_with_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.AssociateProject`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithReadHeaders`

- AssociateProject sends PUT OS-EP-FILTER/endpoint_groups/{id}/projects/{project_id} without query or body, accepts every status below 400 and returns 4xx with AssociateProject operation context.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupProjectAssociation`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/create_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithCreateHeaders`

- Create POSTs OS-EP-FILTER/endpoint_groups with only the declared EndpointGroup attributes under endpoint_group (explicit null kept, unknown dropped, empty body allowed), accepts every status below 400, overlays the endpoint_group response on the request seed and returns 4xx with Create operation context.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupCreateSendsDeclaredAttrs`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/delete_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithDeleteHeaders`

- Delete sends DELETE OS-EP-FILTER/endpoint_groups/{id}, ignores a clean 404 by default like ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and returns other 4xx/5xx with Delete operation context.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupDeleteIgnoreMissing`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/disassociate_project_from_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.DisassociateProject`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithReadHeaders`

- DisassociateProject sends DELETE OS-EP-FILTER/endpoint_groups/{id}/projects/{project_id} without query or body, accepts every status below 400 and returns 404 and other 4xx with DisassociateProject operation context because Python has no ignore_missing here.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupProjectAssociation`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/endpoint_group_endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.ListEndpoints`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListHeaders`

- ListEndpoints sends GET OS-EP-FILTER/endpoint_groups/{id}/endpoints with interface, region_id, service_id, limit and marker as wire query, filters id/name/is_enabled/links/url locally, follows an explicit limit with last-ID marker requests until an empty page and projects EndpointGroupEndpoint rows with is_enabled converted from enabled and endpoint_group_id set.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupScopedLists`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers, unlike Python True == 1.

### `python:identity/v3/endpoint_group_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListHeaders`

- ListProjects sends GET OS-EP-FILTER/endpoint_groups/{id}/projects with the Project QueryParameters transposed (is_enabled as enabled=True, any_tags as repeated tags-any, domain_id), filters id/description/options/links locally, keeps unknown project keys, converts is_enabled/is_domain/options/tags and sets endpoint_group_id on each row.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupScopedLists`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers, unlike Python True == 1.

### `python:identity/v3/endpoint_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListName`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListOpts`

- List sends GET OS-EP-FILTER/endpoint_groups with Accept application/json and only name as wire query, drops limit and marker because EndpointGroup excludes the pagination defaults, applies id/description/filters/links as local Body filters, turns max_items into limit with last-ID marker continuation, reads one page when paginated is false and follows top-level next.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupListQueryFiltersAndPaging`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers, unlike Python True == 1.

### `python:identity/v3/endpoint_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.ListEndpointProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListHeaders`

- ListEndpointProjects sends GET OS-EP-FILTER/endpoints/{id}/projects with the Project QueryParameters (name, marker) as wire query and sets endpoint_id on each EndpointProject view with is_domain converted.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupScopedLists`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers, unlike Python True == 1.

### `python:identity/v3/find_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithFindIgnoreMissing`

- Find sends GET OS-EP-FILTER/endpoint_groups/{x}; after a clean 400, 403 or 404 it lists OS-EP-FILTER/endpoint_groups?name={x} and matches exact ID or name, returning nil by default, resource.NotFoundError with WithFindIgnoreMissing(false) and resource.AmbiguousError for duplicates, while a 500 does not fall back.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupFindByIDThenName`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Python raises DuplicateResource and NotFoundException; Go returns resource.AmbiguousError and resource.NotFoundError wrapped with Find operation context.

### `python:identity/v3/get_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithReadHeaders`

- Get sends GET OS-EP-FILTER/endpoint_groups/{id} with the ID escaped as one segment, accepts every status below 400 with the ID seeded into the view, keeps a non-object response as ResponseError proof and returns 4xx/5xx as ErrUnexpectedResponseCode with Get operation context.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupGetStatusesAndEscapedID`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/project_endpoint_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.ListProjectEndpointGroups`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListName`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListHeaders`

- ListProjectEndpointGroups sends GET OS-EP-FILTER/projects/{id}/endpoint_groups with only name as wire query (limit dropped like ProjectEndpointGroup's QueryParameters) and sets project_id on each EndpointGroup view.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupScopedLists`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers, unlike Python True == 1.

### `python:identity/v3/update_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithUpdateHeaders`

- Update PATCHes OS-EP-FILTER/endpoint_groups/{id} with only declared non-id attributes under endpoint_group, overlays the response on the seeded request values, sends nothing when no declared attribute is dirty and returns 4xx with Update operation context.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupUpdatePatchesDirtyAttrs`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.
- Go header options are an explicit extension; SDK-owned auth, host and connection headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:cloud/_identity/IdentityCloudMixin/create_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.CloudCreate`

- CloudCreate POSTs OS-EP-FILTER/endpoint_groups with name and filters always and description only when nonempty, returning the created record or the 4xx with CloudCreate operation context.
  Tests: `identity/v3/endpointgroups/python_cloud_test.go:TestPythonCloudEndpointGroupCreate`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.

### `python:cloud/_identity/IdentityCloudMixin/update_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithUpdateField`

- The cloud update delegates to the proxy update; Update PATCHes OS-EP-FILTER/endpoint_groups/{id} with the declared name, filters and description values under endpoint_group.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupUpdatePatchesDirtyAttrs`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Python valid_kwargs rejects keyword arguments other than name, filters and description before HTTP; Go Update follows the proxy rule and drops unknown attributes, so the caller chooses which WithUpdateField values to pass.
- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.

### `python:cloud/_identity/IdentityCloudMixin/list_endpoint_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithListName`

- The cloud list delegates to the proxy list; List sends GET OS-EP-FILTER/endpoint_groups with name as the only wire query.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupListQueryFiltersAndPaging`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Python valid_kwargs accepts only name; Go List also accepts the proxy query keywords.
- Go returns a lazy iterator; the Python cloud helper returns a materialized list.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.

### `python:cloud/_identity/IdentityCloudMixin/get_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.Find`

- The cloud get is find_endpoint_group(name_or_id=id) with ignore_missing True; Find sends GET by ID, falls back after 400/403/404 to the name-filtered list and returns nil when nothing matches.
  Tests: `identity/v3/endpointgroups/python_records_test.go:TestPythonEndpointGroupFindByIDThenName`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes an ID string, escapes it as exactly one path segment and returns an owned Record whose Resource.Body is the attribute-named to_dict view with location null instead of the Connection location.

### `python:cloud/_identity/IdentityCloudMixin/delete_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.CloudDelete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithReadHeaders`

- CloudDelete runs Find with ignore_missing, returns false without DELETE when nothing matches, otherwise DELETEs the literal id argument with the 404-ignoring default and returns true; a failed DELETE returns false.
  Tests: `identity/v3/endpointgroups/python_cloud_test.go:TestPythonCloudEndpointGroupDelete`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Python logs a failed DELETE and returns False; Go returns false together with the error wrapped with CloudDelete operation context.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.

### `python:cloud/_identity/IdentityCloudMixin/search_endpoint_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.API.CloudSearch`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups.WithSearchHeaders`

- CloudSearch reads the complete unfiltered GET OS-EP-FILTER/endpoint_groups, then applies _utils._filter_list on the attribute views: exact or glob ID/name selection, nested dictionary filters that fail on a missing key, and the deprecated JMESPath expression returning arbitrary JSON; list errors keep operation context.
  Tests: `identity/v3/endpointgroups/python_cloud_test.go:TestPythonCloudEndpointGroupSearch`.

Python/Go differences:

- Python cloud helpers first call utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 client given to endpointgroups.New without that check.
- Go returns SearchResult with the selected JSON Value, the selected records (nil for an expression) and the full Inventory; Python returns the list or the JMESPath result.
- The row views carry location null, so a filter on location sees null instead of the Python Connection location.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
group = conn.identity.create_endpoint_group(name="web", filters={"interface": "public"})
conn.identity.associate_project_with_endpoint_group(group, project_id)
for endpoint in conn.identity.endpoint_group_endpoints(group):
    print(endpoint.url)
conn.identity.delete_endpoint_group(group)
```

```go
api := endpointgroups.New(identityClient)
group, err := api.Create(ctx,
    endpointgroups.WithCreateField("name", "web"),
    endpointgroups.WithCreateField("filters", map[string]string{"interface": "public"}))
if err != nil {
    return err
}
id := group.Text("id")
if err := api.AssociateProject(ctx, id, projectID); err != nil {
    return err
}
for endpoint, err := range api.ListEndpoints(ctx, id) {
    if err != nil {
        return err
    }
    fmt.Println(endpoint.Text("url"))
}
return api.Delete(ctx, id)
```

## References

- [`projectendpoints`](../projectendpoints)
