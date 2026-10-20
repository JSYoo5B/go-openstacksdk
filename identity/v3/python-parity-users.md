# Python Parity Users

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/add_user_to_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AddToGroup`

- AddToGroup sends PUT groups/{group}/users/{user} without a body like Python add_user_to_group and returns the HTTP error for 404.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserGroupMembership`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Go takes the group ID first and the user ID second, the reverse of the Python argument order.
- Python raises only for status 400 and above; the native call accepts only 204.

### `python:identity/v3/assign_domain_role_to_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`

- roles.API.Assign with DomainID and GroupID sends PUT domains/{id}/groups/{id}/roles/{role} like Python assign_domain_role_to_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Assign sends PUT OS-INHERIT/domains/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Assign instead of roles.API.Assign.

### `python:identity/v3/assign_domain_role_to_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`

- roles.API.Assign with DomainID and UserID sends PUT domains/{id}/users/{id}/roles/{role} like Python assign_domain_role_to_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Assign sends PUT OS-INHERIT/domains/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Assign instead of roles.API.Assign.

### `python:identity/v3/assign_project_role_to_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`

- roles.API.Assign with ProjectID and GroupID sends PUT projects/{id}/groups/{id}/roles/{role} like Python assign_project_role_to_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Assign sends PUT OS-INHERIT/projects/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Assign instead of roles.API.Assign.

### `python:identity/v3/assign_project_role_to_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`

- roles.API.Assign with ProjectID and UserID sends PUT projects/{id}/users/{id}/roles/{role} like Python assign_project_role_to_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Assign sends PUT OS-INHERIT/projects/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Assign instead of roles.API.Assign.

### `python:identity/v3/assign_system_role_to_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`

- roles.API.Assign with System and GroupID sends PUT system/groups/{id}/roles/{role} like Python assign_system_role_to_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python ignores the system argument value and always uses /system; Go sets System: true.

### `python:identity/v3/assign_system_role_to_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`

- roles.API.Assign with System and UserID sends PUT system/users/{id}/roles/{role} like Python assign_system_role_to_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python ignores the system argument value and always uses /system; Go sets System: true.

### `python:identity/v3/check_user_in_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.IsMemberOfGroup`

- IsMemberOfGroup sends HEAD groups/{group}/users/{user}, returns true for 204 and false without error for 404 like Python check_user_in_group, and returns an error for 403.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserGroupMembership`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Go takes the group ID first and the user ID second, the reverse of the Python argument order.
- Python returns False for any other status below 400 such as 200; the native call returns an error for every status except 204 and 404.

### `python:identity/v3/create_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationLocation`

- Legacy native Create sends POST domains with the {"domain": {...}} envelope and decodes native model fields; Domain.options is omitted from that native model, rather than representing every Python Domain field.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Owned CreateRecord sends only explicit declared raw Domain attributes in a domain POST envelope without requiring name or inventing descriptor defaults. Last input alias wins; request enabled/options values are not cast, while returned Domain Resource projects bool/dict/null/defaults and original response evidence retains unknown fields.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainCreateRecordNormalizesAliasesWithoutSendingDefaults`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainCreateRecordDistinguishesOmittedAndExplicitNull`.
- Options/raw values/headers/version/location are independently owned through callbacks; nil/invalid/source-changing/cancel options fail safely. Rejected404 physical read/Close/cancel failures preserve their causes and close once; clean404 retains actual native status/body proof. None becomes lookup fallback or successful no-op. Source-changing callbacks are separately rejected in preflight before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsOwnOptionsHeadersVersionAndLocation`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPhysicalFaultsAndRejectionsDoNotFallback`.

Python/Go differences:

- Legacy native Create returns a Gophercloud model pointer rather than the mutable Python Resource; the new sibling CreateRecord returns an independently owned DomainRecord with options and original response evidence.
- Legacy native Create/Update: Python accepts any status below 400; the native Create accepts only 201.
- Legacy native Create/Update: domains.CreateOpts requires Name, so Go rejects before HTTP a Python create_domain without a name.
- Legacy native Create/Update: Domain options have no typed CreateOpts field and are sent with domains.WithCreateField("options", ...).
- New owned mutation APIs leave native Create/Update untouched. They take finite UTF-8 JSON attributes and safe string IDs in a freshly seeded Domain operation, return independent DomainRecord views plus physical Wire/Envelope/header/status, retain Domain.options and explicit null and drop unknown Body attributes. Request values remain raw; descriptor casts apply only to the returned view.
- The fixed Domain collection/member route deliberately rejects raw base_path with ErrUnsupported/ErrInvalidOption rather than reproducing Source arbitrary URI overrides or formatter/runtime objects. Headers, Microversion and Location are explicit Go conveniences; configured location is copied, unknown server project_id/location is not trusted. Class/session/adapter injection and caller custom transport cache bypass are not claimed.
- Ordered attribute helpers use last canonical/wire alias occurrence; bulk Attributes maps use sorted keys as a deterministic Go adaptation. Mutable existing Resource baseline/equality/recommit state is not modeled. SDK-R1/C1/S1, authenticated cloud acceptance and release verification remain separately tracked whole-SDK goals.

### `python:identity/v3/create_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithCreateField`

- Create sends POST groups with the {"group": {...}} envelope that Python create_group sends and decodes the fields Python exposes.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Create accepts only 201.
- groups.CreateOpts requires Name, so Go rejects before HTTP a Python create_group without a name.
- Python attributes without a typed field go into groups.CreateOpts.Extra or WithCreateField inside the group envelope.

### `python:identity/v3/create_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithCreateField`

- Create sends POST projects with the {"project": {...}} envelope that Python create_project sends and decodes the fields Python exposes.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Create accepts only 201 and 202.
- projects.CreateOpts requires Name, so Go rejects before HTTP a Python create_project without a name.
- Python attributes without a typed field go into projects.CreateOpts.Extra or WithCreateField inside the project envelope.
- Project allows unknown attributes in the Python body; Go sends them through CreateOpts.Extra, and the native Project keeps links and unknown response keys in Extra.

### `python:identity/v3/create_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithCreateField`

- Create sends POST roles with the {"role": {...}} envelope that Python create_role sends and decodes the fields Python exposes.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Create accepts only 201.
- roles.CreateOpts requires Name, so Go rejects before HTTP a Python create_role without a name.
- Python attributes without a typed field go into roles.CreateOpts.Extra or WithCreateField inside the role envelope.

### `python:identity/v3/create_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithCreateField`

- Create sends POST users with the {"user": {...}} envelope that Python create_user sends and decodes the fields Python exposes.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Create accepts only 201.
- users.CreateOpts requires Name, so Go rejects before HTTP a Python create_user without a name.
- Python attributes without a typed field go into users.CreateOpts.Extra or WithCreateField inside the user envelope.
- The native User decodes email and federated into Extra and parses password_expires_at as a time without a zone, so a value ending in Z is a decode error where Python keeps the string.

### `python:identity/v3/delete_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE domains/{id}, succeeds on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and keeps a 409 as an error.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE groups/{id}, succeeds on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and keeps a 409 as an error.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE projects/{id}, succeeds on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and keeps a 409 as an error.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE roles/{id}, succeeds on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and keeps a 409 as an error.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/delete_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE users/{id}, succeeds on 404 like the Python ignore_missing=True default, returns resource.ErrNotFound with WithMissingError like ignore_missing=False, and keeps a 409 as an error.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v3/domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.DomainRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListLocation`

- List sends the single GET domains that Python domains() sends without arguments, maps the Python query names name and enabled through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainListRequests`.
- Source Domain declares raw description/links/id/name, nullable CPython bool is_enabled from enabled and dict options (nonobject {}, null preserved), unknown Body discarded from declared view but complete Wire/Envelope/header/status retained; canonical server query alias is_enabled wins enabled presence and unknown kwargs are dropped before marshal.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsDeclaredDescriptorsQueryAliasesAndOwnedProof`.
- Pinned root links dictionary is ignored with links-key presence precedence, explicit null links is an iterable error with actual page proof, and short limit pages continue by raw last ID until empty. Raw max_items precedes filtering; break stops lookahead.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsSourceDictionaryLinksAndPresence`, `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsFilteredMarkerRawCapAndBreak`.
- Ordered recursive local filters use actual truthiness and CPython loaded float/bool/integer equality; early mismatch hides later bad shape and reached bad shape keeps physical proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsOrderedLoadedLocalEquality`.
- Proxy expressions see an opaque unread generator, with HTTP0 null/type/literal results, identity/nested generator preservation, no invented JSON and typed InvalidOption cause for syntax/type failure; falsey/nonstrings remain lazy.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Proxy formal headers/version/basepath and one-pass conflicting attrs normalization restore query/headers/version, retain original expression priority and unknown discard; bound control collisions remain HTTP0.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsProxyFormalRestorationAndBoundCollision`.
- Explicit location is owned and ignores unknown wire project/location; new SourceStatusPolicy accepts valid HTTP100..399 without changing older profiles/native statuses or triggering discovery.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsLocationBeforeLocalFilterAndStatusPolicy`.
- Captured options/Source/context guard each callback before and after; restoring callback cannot undo observed source mutation or custom cancellation and invalid preflight makes HTTP0/no proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsOwnedOptionsStickySourceAndCancellation`.
- 28 accepted/rejected physical Read/Close/Source restored by Close/custom cancel lanes preserve actual proof, terminal sticky causes, Close once and no retry/backoff/reauth. Native pointer401 has separately inspected value ErrOriginal and ErrReauth; no nonexistent Unwrap is assumed.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsFaultedActualRejectionsNeverFallbackOrRetry`.
- Clean native401/503/429/498 retries reuse live token and owned route/query/headers with bodyless GET, preserving successful HTTP203 proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

### `python:identity/v3/get_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.DomainRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetLocation`

- Get sends GET domains/{id} like Python get_domain and decodes the domain envelope.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Literal member GET uses existing private Domain Source profile/converter with six nullable raw/bool/dict descriptors, preserves options missing/null/nonobject/object and constructor ID seed/explicit response null; unknown response remains Wire and independent Envelope/header/status/Clone records actual100/199/200/203/399.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordNullableRawOptionsSeedAndActualEvidence`.
- Direct rejected403/404 never falls back; actual native response proof retains method/URL/status/body/header. Accepted malformed singular envelope keeps actual ResponseError.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordRejectedNeverFallsBackAndBadEnvelopeKeepsProof`.
- Transport/location options snapshot maps/pointers, retain configured project despite unknown response scope, and per-callback selected source/context/cancel checks prevent restore callbacks and HTTP0 fake receipts.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordOwnedOptionsLocationAndStickyGuard`.
- 28 physical accepted/rejected Read/Close/source restored on Close/custom cancel lanes retain immutable actual proof, Close once and zero Retry/Backoff/Reauth; native pointer401 ErrOriginal value and ErrReauth graph are separately checked. Clean401/503/429/498 uses live token and owned bodyless request.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordPhysicalFaultsKeepActualProofAndNoRetry`, `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordCleanRetryPreservesLiveAuthAndFrozenRequest`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.
- SDK-owned typed Go transport/location conveniences and raw nullable Domain record preserve actual Wire/Envelope/header/status; existing native Domain Get/Create/Update and full find_domain judgments stay unchanged.
- Go uses required literal nonblank UTF-8 string identity, immutable escaped path and selected native-v3 client. Source empty/numeric/false IDs may request domains/ collection; mutable Resource/dict/object inputs and unsafe raw URL composition are narrowed to documented Go/SDK-C1/S1 boundaries.
- Direct Source actual both-None finder raises no-ID InvalidRequest despite misleading cloud comment; Go ErrInvalidOption preflight preserves HTTP0 instead of pretending None success.
- Ignored filters are not evaluated. Source warnings.warn is represented by DeprecatedFilters presence flag; global Python warning-as-error configuration is not cloned.
- Explicit immutable location is a Go convenience and default standalone view is null; automatic Connection bridge/current scope inference is absent.
- Generic cache/synchronized arbitrary Resource lifecycle/session/Connection remain SDK-R1/C1/S1 differences; finite named get implementation is not full SDK supported. No arbitrary fetch controls or full Proxy.find_domain audit/row promotion.

### `python:identity/v3/get_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Get`

- Get sends GET groups/{id} like Python get_group and decodes the group envelope.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.

### `python:identity/v3/get_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Get`

- Get sends GET projects/{id} like Python get_project and decodes the project envelope.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.

### `python:identity/v3/get_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Get`

- Get sends GET roles/{id} like Python get_role and decodes the role envelope.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.

### `python:identity/v3/get_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Get`

- Get sends GET users/{id} like Python get_user and decodes the user envelope.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.

### `python:identity/v3/group_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListInGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AllUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ProxyUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.UserRecord.Clone`

- ListInGroup sends GET groups/{id}/users with the User query mapping such as password_expires_at like Python group_users and sends a limit without a marker follow-up.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserGroupUsersRequests`.
- group_users uses complete User profile, not reduced subclass: declared query incl federated IDs/expiration/enabled alias and local email/Body preserve raw dates/defaults/casts/unknown discard with literal escaped fixed Group collection and actual private proof.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsFullUserProfileAndFixedGroupScope`.
- Immutable group route persists across short-limit raw cursor pages, filtering/cap/break and caller view mutations; Source root dictionary/null and cross-group continuation guard keep original actual page proof.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsMarkerRawCapAndBreakKeepGroupRoute`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsSourceLinksAndCrossGroupContinuationProof`.
- Source-bound base_path/null/raw group collision and restored duplicate controls fail before HTTP; malformed literal ID and bool limit use documented safe Go validation without fabricated response.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsBasePathBindingAndBooleanLimitPreflight`.
- Opaque group Proxy uses original User generator model without HTTP for projection/type errors and preserves lazy identity; group-specific profile copy never changes subsequent standalone User list path.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsOpaqueProxyAndStandaloneProfileIsolation`.
- Loaded nested User Body subset/equality and explicit immutable User location retain configured scope despite default_project_id/unknown project_id/location. Full User schema/header/version formal restore is SDK-owned.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsLoadedLocalFilterAndExplicitLocation`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsFormalRestorationUsesFullUserSchema`.
- Option map/raw/pointer snapshots and sticky before/after selected source/custom cancel guard prevent restoration callbacks and physical requests on invalid preflight.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsOptionsCloneStickySourceAndCancel`.
- 28 physical accepted/rejected Read/Close/restored Source/cancel lanes preserve actual body/header/status and independent value ErrOriginal/native pointer401 ErrReauth, Close once and no fault retry; clean401/503/429/498 uses live token and frozen group request.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsPhysicalFaultsKeepActualProofAndStopRetry`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsCleanRetryKeepsGroupScopeAndLiveAuth`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Library-owned concrete functional options, raw nullable Source view and separate actual receipt preserve existing native APIs. No user builder interface or new transport framework.
- Go caps/limit require nonnegative decimal integers. Source raw limit True/False is sent as True/False and truthy True activates fallback; Go rejects bool before HTTP and does not substitute1/0. Safe fixed origin/path/query/header/cursor/cycle bounds are explicit.
- Source rootlink JSON precedence/dictionary-ignore/null-error is preserved. RFC HTTP Link uses explicit Go interoperability for ordinary Requests next[url] versus Source next[uri] bug.
- Unconsumed Proxy generator is opaque and no fake JSON/HTTP association is created. Go iter.Seq2 reuse reissues captured requests instead of sharing Python one-shot state.
- Standalone default location is null. Explicit cloned recorded facts are a Go convenience; Region has no zone descriptor and normalizes zone null, while GroupUser reuses existing User explicit-zone convenience. Unknown response scope fields are not trusted.
- Already selected Keystone v3 native API required. Mutable Resource object/group subclass routing, automatic Connection location/cache, generic synchronized lifecycle and session remain SDK-R1/C1/S1.
- Group identity is a required literal escaped segment; Source Group object base_path changes, slash path composition and percent URI formatting failures are bounded rather than copied. New Group-to-User names avoid existing inverse User-to-Group ListGroupRecords. Source base_path keyword presence incl null is rejected as already bound.
- Generic SDK-R1/C1/S1 response cache, arbitrary synchronized Resource/object lifecycle and automatic Connection/session bridge are not completed. These two finite mappings have supported delta0; root Go/race/main gates must pass before catalog promotion.

### `python:identity/v3/groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.GroupRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListLocation`

- List sends the single GET groups that Python groups() sends without arguments, maps the Python query names domain_id and name through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupListRequests`.
- Lazy Group records classify limit/marker/domain_id/name as server query, description/id as ordered local Body filters, discard unknown before marshaling and keep nullable raw Body separate from original independent Wire/Envelope/header/status.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsLazyUntypedDeclaredViewAndEvidence`.
- Source root links dictionary is ignored with present-key precedence over groups_links; explicit limit continues even short nonempty pages by last raw wire ID until empty. Local filtering does not change marker, raw max_items precedes filtering and break has no lookahead. Explicit root links:null is a Source iterable error and preserves actual page proof.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsSourceDictionaryLinksAndPresence`, `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsFilteredMarkerRawCapAndBreak`.
- Loaded local equality preserves CPython float rounding/underflow and bool/integer equality, ordered recursive subset/actual truthiness, mismatch short circuit and actual proof for reached bad shapes.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsOrderedLoadedLocalEquality`.
- Proxy JMESPath receives opaque unconsumed generator: projection/type/literal JSON HTTP0; @ keeps lazy generator; nested generator has no fabricated JSON; type/invalid expression does not request and falsey/nonstring remains lazy.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Owned request headers/microversion/base_path formal arguments and one-pass __conflicting_attrs restoration preserve restored query/header/version, original JMESPath priority, unknown attr discard and HTTP0 bound collisions.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsProxyFormalRestorationAndBoundCollision`.
- Immutable options plus sticky Source/context guards run before and after every callback; source-change then restore callback cannot escape and bad preflight creates HTTP0 without fabricated ResponseError.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsOwnedOptionsStickySourceAndCancellation`.
- Physical Read/Close/sticky Source restored by Close/custom cancel matrix for200/400/403/404/401/429/503 stops Retry/Backoff/Reauth/fallback with actual body/header/status and pointer401 wrapper and separately inspected ErrOriginal plus ErrReauth. Accepted200 keeps ResponseError; rejected400+ keeps native ErrUnexpectedResponseCode proof and pointer401 separates ErrOriginal/ErrReauth.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsFaultedActualRejectionsNeverFallbackOrRetry`.
- Clean native401/503/429/498 uses live provider token and captured header/route/query with bodyless GET; successful result retains actual HTTP203.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- groups.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Concrete Go options own request/defaults, nullable raw Resource and independent actual response channels. Existing native List/Get/FindIdentity semantics are unchanged.
- Already selected Keystone v3 API required; no automatic v2 ensure_service_version conversion or Connection current location inference. Explicit immutable CloudLocation retains current project because Group declares no project_id/zone.
- Fixed origin/collection/query/header/version bounds reject foreign URL/traversal/cycle and malformed identities/markers using actual receipt. Paging amounts are nonnegative Go integers.
- iter.Seq2 can be iterated again and repeats captured requests rather than Python generator single-consumption state. Proxy pure expressions still perform SDK option/source preflight before returning.
- Connection cache and generic synchronized Resource object lifecycle remain separately tracked as SDK-R1/C1; finite Group read mapping is not full SDK parity.
- RFC HTTP Link is deliberate Go interoperability: Source requests next[uri] does not reproduce the ordinary Requests next[url] shape; root Source-link JSON precedence is exact.

### `python:identity/v3/projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListJMESPath`

- List sends the single GET projects that Python projects() sends without arguments, maps the Python query names domain_id, enabled, is_domain, name, parent_id, tags, tags-any, not-tags and not-tags-any through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectListRequests`.
- Default bodyless /projects GET, selected identity role, Source Body aliases/casts/defaults and explicit-null presence retain independent Resource/Wire/page/header/status evidence including large integer extensions.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsDeclaredViewKeepsPhysicalEvidenceAndPresence`.
- Server canonical/wire QueryParameters use Requests boolean/repeated/null values; declared non-query Body filters use ordered nested CPython loaded equality, empty-dict truthiness and unknown-query discard before marshaling.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsQueryAliasesAndOrderedLoadedLocalFilters`.
- Truthy limit repeats even short pages without links using the last raw row marker, raw cap counts locally excluded rows, empty page ends and consumer break stops before another request.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsLimitMarkerWithoutLinkRawCapAndBreak`.
- Pinned Keystone dictionary-next is ignored; dangerous advertised top-level origin/filter changes and cycles preserve the actual whole page receipt.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsKeystoneDictionaryNextAndRouteGuards`.
- Source opaque Proxy JMESPath runs before HTTP and preserves null/literal values, generator identity and nested unconsumed generators; typed list remains resource-only.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsProxyOpaqueJMESIsUnreadAndRetainsGenerator`.
- DTO/functional options snapshot values and headers; nil context/options, selected source mutation and custom cancellation fail before physical HTTP without fabricated receipt.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsOptionsSnapshotAndSourceCancellationPreflight`.
- Malformed plural/null/colliding row and nonmapping local filter failures retain accepted page status/body/header. Clean native reauthentication keeps identity version and live token.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsInvalidRepresentationRetainsActualHeaderAndPage`, `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsCleanReauthenticationAndVersionHeaderAreIdentityOwned`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- projects.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.

### `python:identity/v3/remove_user_from_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.RemoveFromGroup`

- RemoveFromGroup sends DELETE groups/{group}/users/{user} like Python remove_user_from_group and returns the HTTP error for 404.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserGroupMembership`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Go takes the group ID first and the user ID second, the reverse of the Python argument order.
- Python raises only for status 400 and above; the native call accepts only 204, and a 404 is an error in both.

### `python:identity/v3/role_assignments_filter`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListAssignmentsOnResource`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsOnResourceOptions`

- ListAssignmentsOnResource sends the single GET {domains|projects}/{id}/{users|groups}/{id}/roles or system/{users|groups}/{id}/roles that Python role_assignments_filter sends for each target and actor, decodes role id and name, and rejects two targets before HTTP.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignmentsFilterRoutes`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python raises InvalidRequest only when domain, project and system are all given and otherwise picks domain, then project, then system; Go requires exactly one target and rejects the rest before HTTP.
- Python ignores the system value and always uses /system; Go takes System as a bool.
- Python returns RoleXAssignment Resources with the target and actor IDs set from the URI; Go returns roles.Role values without them.
- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.

### `python:identity/v3/roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.RoleRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListJMESPath`

- List sends the single GET roles that Python roles() sends without arguments, maps the Python query names domain_id and name through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleListRequests`.
- Bodyless default roles GET and explicit SourceStatusPolicy accept actual199. Six Role Body descriptors/default/null/options-dict cast discard unknowns only from declared view and retain all actual Wire/page/header/status evidence; Clone channels are independent.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence`.
- Only name/domain_id/limit/marker go to the server with Requests repeated and bool/null values. Ordered local description/options/links/id filters use CPython loaded equality and recursive subset, empty-dict matches truthy scalar/list, and unknown kwargs discard before marshal.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsDeclaredQueryAndOrderedLoadedLocalFilters`.
- Explicit positive limit repeats short nonempty pages without links with last raw ID; local-excluded rows count in marker/raw cap; break stops before another request and reusable iterator repeats guarded captured calls.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsRawMarkerShortPagesCapBreakAndReusableIterator`.
- Source Keystone dictionary-next is ignored, root links presence masks plural links, root null/foreign URL/query change/cycle retain actual whole page proof.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsSourceDictionaryRootPresenceAndActualContinuationGuards`.
- Unconsumed Proxy generator JMESPath returns null/literals/opaque identity/nested generator before HTTP; array type errors do not consume it. Known formal options, snapshots, reserved/collision/unsafe path/cap validation, source mutation and custom callback cancel fail without fake response.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsOptionSnapshotsFormalsAndPreflightSourceGuards`.
- Immutable typed location retains recorded current scope, ignores unknown project_id, and physical malformed rows/nonmapping local filter preserve accepted203 page.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage`.
- Accepted/rejected200/400/403/404/401/429/498/503 read/Close/both/restored-source+custom-cancel matrix keeps one physical HTTP/Close, actual native or accepted proof, all causes and zero retry/backoff/reauth. Clean401/503/429/498 preserve fixed request/version/absent conditions and live renewed token.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- roles.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.

### `python:identity/v3/unassign_domain_role_from_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`

- roles.API.Unassign with DomainID and GroupID sends DELETE domains/{id}/groups/{id}/roles/{role} like Python unassign_domain_role_from_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Unassign sends DELETE OS-INHERIT/domains/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Unassign instead of roles.API.Unassign.

### `python:identity/v3/unassign_domain_role_from_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`

- roles.API.Unassign with DomainID and UserID sends DELETE domains/{id}/users/{id}/roles/{role} like Python unassign_domain_role_from_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Unassign sends DELETE OS-INHERIT/domains/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Unassign instead of roles.API.Unassign.

### `python:identity/v3/unassign_project_role_from_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`

- roles.API.Unassign with ProjectID and GroupID sends DELETE projects/{id}/groups/{id}/roles/{role} like Python unassign_project_role_from_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Unassign sends DELETE OS-INHERIT/projects/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Unassign instead of roles.API.Unassign.

### `python:identity/v3/unassign_project_role_from_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`

- roles.API.Unassign with ProjectID and UserID sends DELETE projects/{id}/users/{id}/roles/{role} like Python unassign_project_role_from_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Unassign sends DELETE OS-INHERIT/projects/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Unassign instead of roles.API.Unassign.

### `python:identity/v3/unassign_system_role_from_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`

- roles.API.Unassign with System and GroupID sends DELETE system/groups/{id}/roles/{role} like Python unassign_system_role_from_group, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python ignores the system argument value and always uses /system; Go sets System: true.

### `python:identity/v3/unassign_system_role_from_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`

- roles.API.Unassign with System and UserID sends DELETE system/users/{id}/roles/{role} like Python unassign_system_role_from_user, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python sends the request through Proxy.request with raise_exc=False and discards the response, so assign and unassign return None for every HTTP status; the native call accepts only 204 and returns other statuses as errors.
- Python ignores the system argument value and always uses /system; Go sets System: true.

### `python:identity/v3/update_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationLocation`

- Legacy native Update sends PATCH domains/{id} with the {"domain": {...}} envelope and optional explicit WithUpdateField("id", id); the old fixture documents that opt-in native body, not pinned Resource.commit dirty-id behavior.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Owned scalar-ID UpdateRecord sends PATCH with explicit raw declared attributes including nullable description/enabled/options; constructor ID is seeded but never dirty or sent. Existing response overlays seed and retains options plus physical response proof; duplicate attrs id is rejected before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordPATCHPreservesSeedAndActualResponse`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordExplicitNullIsSentAsDirty`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`.
- Empty or only-unknown scalar-ID updates return a freshly projected ID seed with no HTTP, Wire or fabricated status. This is distinct from mutable Python Resource equality/recommit, and explicit null remains dirty and sends a PATCH.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordEmptyOrUnknownAttrsAvoidHTTP`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordExplicitNullIsSentAsDirty`.
- Options/raw values/headers/version/location are independently owned through callbacks; nil/invalid/source-changing/cancel options fail safely. Rejected404 physical read/Close/cancel failures preserve their causes and close once; clean404 retains actual native status/body proof. None becomes lookup fallback or successful no-op. Source-changing callbacks are separately rejected in preflight before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsOwnOptionsHeadersVersionAndLocation`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPhysicalFaultsAndRejectionsDoNotFallback`.

Python/Go differences:

- Legacy native Create/Update: Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Legacy native Create/Update: Python accepts any status below 400; the native Update accepts only 200.
- Legacy native Update permits an explicitly added id body field through WithUpdateField("id", id). The pinned Resource.commit actually discards dirty id before preparing PATCH; the new owned UpdateRecord follows that behavior instead of the older fixture comment.
- New owned mutation APIs leave native Create/Update untouched. They take finite UTF-8 JSON attributes and safe string IDs in a freshly seeded Domain operation, return independent DomainRecord views plus physical Wire/Envelope/header/status, retain Domain.options and explicit null and drop unknown Body attributes. Request values remain raw; descriptor casts apply only to the returned view.
- The fixed Domain collection/member route deliberately rejects raw base_path with ErrUnsupported/ErrInvalidOption rather than reproducing Source arbitrary URI overrides or formatter/runtime objects. Headers, Microversion and Location are explicit Go conveniences; configured location is copied, unknown server project_id/location is not trusted. Class/session/adapter injection and caller custom transport cache bypass are not claimed.
- Ordered attribute helpers use last canonical/wire alias occurrence; bulk Attributes maps use sorted keys as a deterministic Go adaptation. Mutable existing Resource baseline/equality/recommit state is not modeled. SDK-R1/C1/S1, authenticated cloud acceptance and release verification remain separately tracked whole-SDK goals.

### `python:identity/v3/update_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithUpdateField`

- Update sends PATCH groups/{id} with the {"group": {...}} envelope including the dirty id that Python update_group sends and decodes the response.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Update accepts only 200.
- Python Proxy._update leaves the id dirty, so the PATCH body carries it; Go adds it with groups.WithUpdateField("id", id) and Keystone ignores it.
- Typed UpdateOpts omit empty strings and nil pointers; a Python None value is sent as null only through the Extra map (for example Extra{"description": nil}).

### `python:identity/v3/update_password`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ChangePassword`

- ChangePassword sends POST users/{id}/password with {"user": {"original_password", "password"}} like Python update_password and returns the HTTP error for 401.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserUpdatePassword`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python raises only for status 400 and above; the native ChangePassword accepts only 204, so a 200 is a Go error.

### `python:identity/v3/update_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithUpdateField`

- Update sends PATCH projects/{id} with the {"project": {...}} envelope including the dirty id that Python update_project sends and decodes the response.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Update accepts only 200.
- Python Proxy._update leaves the id dirty, so the PATCH body carries it; Go adds it with projects.WithUpdateField("id", id) and Keystone ignores it.
- Typed UpdateOpts omit empty strings and nil pointers; a Python None value is sent as null only through the Extra map (for example Extra{"description": nil}).

### `python:identity/v3/update_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithUpdateField`

- Update sends PATCH roles/{id} with the {"role": {...}} envelope including the dirty id that Python update_role sends and decodes the response.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Update accepts only 200.
- Python Proxy._update leaves the id dirty, so the PATCH body carries it; Go adds it with roles.WithUpdateField("id", id) and Keystone ignores it.
- Typed UpdateOpts omit empty strings and nil pointers; a Python None value is sent as null only through the Extra map (for example Extra{"description": nil}).

### `python:identity/v3/update_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithUpdateField`

- Update sends PATCH users/{id} with the {"user": {...}} envelope including the dirty id that Python update_user sends and decodes the response.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400; the native Update accepts only 200.
- Python Proxy._update leaves the id dirty, so the PATCH body carries it; Go adds it with users.WithUpdateField("id", id) and Keystone ignores it.
- Typed UpdateOpts omit empty strings and nil pointers; a Python None value is sent as null only through the Extra map (for example Extra{"description": nil}).

### `python:identity/v3/users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.UserRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListPaginated`

- List sends the single GET users that Python users() sends without arguments, maps the Python query names domain_id, enabled, idp_id, name, password_expires_at, protocol_id and unique_id through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserListRequests`.
- Pinned User nullable declarations/casts/defaults, canonical/wire response last-member order, untyped large password-expiration values, options missing{} vs explicitnull, federated scalar wrapping, unknown discard and independent view/wire/page/header/status/Clone match owned Source policy.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsExactDescriptorDefaultsAndResponseAliasOrder`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLazyQueryAndUntypedView`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsDefaultNullAndNestedLoadedLocal`.
- Mapped domain/name/password expiry/unique/idp/protocol/is_enabled->enabled plus inherited limit/marker preserve Requests repeated/null/bool values and canonical query precedence; declared nonquery Body kwargs run ordered loaded local filters, while unknown kwargs discard before marshal.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsDeclaredQueriesCanonicalWinsRequestsValues`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLoadedLocalFilterAndShortCircuit`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsDefaultNullAndNestedLoadedLocal`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Lazy list defaults, Source root dictionary links ignored with key-presence precedence, short nonempty page positive-limit marker from last raw wire ID, raw prefilter max_items hint/single page/empty terminal and actual unsafe-next proof preserve the named Source list.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsDictionaryLinksIgnoredWithSourcePrecedence`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsFilteredMarkerAndRawMaxItems`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsIteratorKeepsYieldedRowsAndAllDiscards`.
- Copied raw/options/header/MV/base_path and one-pass conflicting attrs, original JMES priority, constructor/formal errors, bound route/context/source guards and explicit100..399 SourceStatusPolicy operate without changing existing native API defaults or inventing discovery.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsSnapshotFormalRestorationAndHeaders`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsSourceSuccessStatusAndHeaderNoDiscovery`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsSingletonMalformedAndConstructorEvidence`.
- Opaque Source Proxy generator receives truthy JMES before HTTP; null/index/projection/type/literals/type-errors and retained/nested generator identity are distinct from materialized arrays. Omitted/falsey/nonstring filters keep lazy rows; typed iterator explicitly directs truthy string callers to polymorphic method.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserProxyListOpaqueProjectionAndRetainedGenerators`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- SourceNoConnection default location null, explicit immutable configured CloudLocation retains current scope because User has no project/zone Body; default_project_id and unknown project_id/location cannot alter it. Raw receipt remains unchanged and invalid location rejects beforeHTTP.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLocationIsExplicitImmutableAndDeclared`.
- Physical200/203/401/404/429/503 read/close/mixed faults preserve actual accepted/native rejection proof plus every cause and native401 ErrOriginal/ErrReauth, oneClose/HTTP and zero Retry/Reauth. Clean401/503 preserve captured query/headers/version and live token; Close cancel preserves real receipt and caller context cause.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsCleanReauthenticationAndRetryKeepPolicy`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsCloseCancellationKeepsReceiptAndCause`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- users.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.

### `python:identity/v3/validate_group_has_domain_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`

- roles.API.Validate with DomainID and GroupID sends HEAD domains/{id}/groups/{id}/roles/{role} like Python validate_group_has_domain_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Validate sends HEAD OS-INHERIT/domains/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Validate instead of roles.API.Validate.

### `python:identity/v3/validate_group_has_project_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`

- roles.API.Validate with ProjectID and GroupID sends HEAD projects/{id}/groups/{id}/roles/{role} like Python validate_group_has_project_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Validate sends HEAD OS-INHERIT/projects/{id}/groups/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Validate instead of roles.API.Validate.

### `python:identity/v3/validate_group_has_system_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`

- roles.API.Validate with System and GroupID sends HEAD system/groups/{id}/roles/{role} like Python validate_group_has_system_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python ignores the system argument value and always uses /system; Go sets System: true.

### `python:identity/v3/validate_user_has_domain_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`

- roles.API.Validate with DomainID and UserID sends HEAD domains/{id}/users/{id}/roles/{role} like Python validate_user_has_domain_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Validate sends HEAD OS-INHERIT/domains/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Validate instead of roles.API.Validate.

### `python:identity/v3/validate_user_has_project_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`

- roles.API.Validate with ProjectID and UserID sends HEAD projects/{id}/users/{id}/roles/{role} like Python validate_user_has_project_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.
- With Python inherited=True, osinherit.API.Validate sends HEAD OS-INHERIT/projects/{id}/users/{id}/roles/{role}/inherited_to_projects.
  Tests: `identity/v3/osinherit/python_parity_test.go:TestPythonInheritedRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python selects the OS-INHERIT route with inherited=True; Go calls osinherit.API.Validate instead of roles.API.Validate.

### `python:identity/v3/validate_user_has_system_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`

- roles.API.Validate with System and UserID sends HEAD system/users/{id}/roles/{role} like Python validate_user_has_system_role, returns nil for 204 and the status error for 404 and 200.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleAssignUnassignValidate`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python returns True only for 204 and False for every other status including 404; the native call returns nil only for 204 and the Gophercloud status error otherwise, so a nil error means True.
- Python ignores the system argument value and always uses /system; Go sets System: true.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
enabled := false
project, err := projects.New(client).Create(ctx, projects.CreateOpts{Name: "demo", DomainID: "default", Enabled: &enabled})
empty := []string{}
project, err = projects.New(client).Update(ctx, "p-1",
    projects.UpdateOpts{Tags: &empty, Extra: map[string]any{"description": nil}},
    projects.WithUpdateField("id", "p-1"))
```

```go
err := users.New(client).Remove(ctx, resource.ID("u-1"))
err = users.New(client).Remove(ctx, resource.ID("u-1"), resource.WithMissingError()) // ignore_missing=False
```

```go
api := roles.New(client)
// Python assign_project_role_to_user("p-1", "u-1", "r-1")
err := api.Assign(ctx, "r-1", roles.AssignOpts{ProjectID: "p-1", UserID: "u-1"})
// Python validate_group_has_system_role("g-1", "r-1", "all")
hasRole := api.Validate(ctx, "r-1", roles.ValidateOpts{System: true, GroupID: "g-1"}) == nil
// Python assign_domain_role_to_group("d-1", "g-1", "r-1", inherited=True)
err = osinherit.New(client).Assign(ctx, "r-1", osinherit.AssignOpts{DomainID: "d-1", GroupID: "g-1"})
```

## References

- [identity v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [Native Admin A](native-admin-a.md)
- [Native Trusts](native-trusts.md)
- [Memberships](users/memberships.md)
- [Finding Identities](../../docs/finding-identities.md)
- [Project](projects/python-record-reads.md)
- [Group](groups/python-records.md)
- [User](users/python_records.md)
- [Python Group Records](users/python-group-records.md)
- [Role](roles/python-records.md)
- [Python Records](domains/python-records.md)
- [Cloud Domain Get Python](../cloud-domain-get-python.md)
- [`create_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L217)
- [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L228)
- [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L278)
- [`domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L290)
- [`update_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L301)
- [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L738)
- [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L749)
- [`find_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L782)
- [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L805)
- [`groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L818)
- [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L829)
- [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L843)
- [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L860)
- [`check_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L877)
- [`group_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L894)
- [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1019)
- [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1030)
- [`find_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1063)
- [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1086)
- [`projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1098)
- [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1150)
- [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1335)
- [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1346)
- [`find_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1379)
- [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1402)
- [`users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429)
- [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1440)
- [`update_password`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1452)
- [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1699)
- [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1710)
- [`find_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1743)
- [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1766)
- [`roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1778)
- [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1788)
- [`role_assignments_filter`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1877)
- [`role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983)
- [`assign_domain_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1999)
- [`unassign_domain_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2023)
- [`validate_user_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2047)
- [`assign_domain_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2070)
- [`unassign_domain_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2094)
- [`validate_group_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2118)
- [`assign_project_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2141)
- [`unassign_project_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2166)
- [`validate_user_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2191)
- [`assign_project_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2215)
- [`unassign_project_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2240)
- [`validate_group_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2265)
- [`assign_system_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2289)
- [`unassign_system_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2309)
- [`validate_user_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2329)
- [`assign_system_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2349)
- [`unassign_system_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2369)
- [`validate_group_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2389)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Python Record Reads](projects/python-record-reads.md)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)
- [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)
- [Groups](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)
- [Domain](domains/python-records.md)
- [Cloud User Role Get Python](../cloud-user-role-get-python.md)
- [User](users/python-record-find.md)
- [Role](roles/python-record-find.md)
- [CreateRecord/UpdateRecord](domains/python-record-mutations.md)
- [Domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/domain.py)
- [Project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py)
- [System](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/system.py)
