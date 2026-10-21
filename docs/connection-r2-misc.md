# Connection-level helpers

These helpers correspond to Python `Connection`/`OpenStackCloud` methods. They live in the root package.

## Global request ID: `Connection.GlobalRequest`

`GlobalRequest(id)` returns a new `Connection` that shares the authenticated provider (token, HTTP client, reauthentication) and sends `X-OpenStack-Request-Id: id` on every request made through services obtained from it (built-in services and `CustomServiceClient`). The receiver and services already obtained from it are unchanged. The ID is fixed for the life of the derived connection: Go has no in-place `set_global_request_id`, so concurrent requests can never observe a change. Derived connections need no `Close`. The ID must be non-empty printable ASCII without spaces; the `req-` prefix honored by services is not enforced. The header is not added to Keystone token requests.

## Endpoint lookup: `EndpointFor`, `SessionEndpoint`, `HasService`

`EndpointFor(type, ...)` returns `(url, found, error)` from a `WithEndpoint` override of that type, else the recorded token catalog (first matching endpoint in catalog order, interface and region as in keystoneauth `url_for`). Region and interface default to the connection's and can be overridden per call with `WithEndpointRegion` and `WithEndpointInterface`; `WithEndpointServiceName` filters on the catalog name. No request is made. Without a recorded token result the provider's endpoint locator is used; without either, `ErrNoAuthResult` is returned. `SessionEndpoint` and `HasService` use the same lookup. Not provided: `min_version`/`max_version` discovery, os-service-types aliases, `has_<service>` configuration flags, and Python's `False` for lookup failures (Go returns the error).

## Re-authentication: `ConnectAs`, `ConnectAsProject`

`ConnectAs(ctx, AsUsername(...), AsProjectName(...), ...)` authenticates again with the options recorded by `Connect` (including values resolved from clouds.yaml) plus typed overrides, and returns an independent connection that inherits endpoint, region, interface, microversion, HTTP/TLS and policy options and the cloud/region location facts. It has its own token, service cache and no global request ID, custom services or shared discovery cache. Python's discard rule is kept: overriding user name or ID discards both, user domain ID/name, and project ID/name. A project selected by name inherits the project domain of the previous name scope, else the user domain, unless `AsProjectDomainID/Name` is given. With a user ID the inherited user domain is dropped because Identity v3 rejects both. `AsAuthOptions` edits the resolved `AuthOptions` directly. `ConnectAsProject(ProjectRef{ID, Name, DomainID})` is the dict form; an ID wins over a name. Connections from `FromProvider` record no credentials and return `ErrNoRecordedAuth`. Authentication happens once per call within the context and is never replayed.

## Custom services: `AddService`, `CustomServiceClient`

`AddService(name, factory [, endpoint])` registers a Gophercloud client factory on one connection (and connections derived with `GlobalRequest` later); built-in names cannot be replaced and registrations cannot change. `CustomServiceClient(ctx, name)` builds the client lazily once and caches it. Python registers on the `Connection` class for the whole process; Go is instance-local.

## Inventory: `Inventory`

`NewInventory`, `NewInventoryFromCloudFiles` and `CloudNames` mirror `OpenStackInventory`. `ListHosts` lists detail servers of each cloud in order (`WithInventoryAllProjects` adds `all_tenants=true`); a cloud contributes nothing if its listing fails. The default aborts at the first failing cloud and returns earlier hosts with the error; `WithInventoryContinueOnCloudError` records failures instead. Cancellation always aborts. `SearchHosts` and `GetHost` apply glob identifier selection and local filters (`WithInventoryFilters`) over the server JSON; `GetHost` returns nil for no match and `resource.AmbiguousError` for several. The Python `expand` interface/floating-IP hostvars and `private` are not applied; `use_direct_get` and cache refresh have no counterpart.

## Not provided: `project_cleanup`

Python `project_cleanup` deletes resources across services in dependency order with filters, dry run and a status queue. No safe, explicit Go contract exists for this destructive multi-service workflow, so it stays unresolved.
