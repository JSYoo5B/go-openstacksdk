# Python Core Miscellaneous Operations

This guide records contracts and support decisions for Python openstacksdk
Connection session accessors and small Python-only utilities, compared against
the pinned source.

## Connection token, user and catalog accessors

`Connection.AuthToken`, `Connection.Authorize`, `Connection.CurrentUserID` and
`Connection.ServiceCatalog` expose facts that the shared Gophercloud provider
already holds.

- `AuthToken` returns the current token string without an HTTP request. It is
  empty when no token is held and never re-authenticates.
- `Authorize(ctx)` returns the current token; when none is held it runs the
  provider reauthentication function once. It does not refresh an expired token
  (a rejected request triggers the native reauthentication). A canceled context
  fails before any call and keeps `context.Canceled` observable; a provider
  with no token and no reauthentication function returns an invalid-option error.
- `CurrentUserID` and `ServiceCatalog` read the recorded Keystone token response
  and make no request. Without a recorded response they return
  `ErrNoAuthResult`. `ServiceCatalog` returns a copy; an absent catalog is an
  empty slice.

Differences from Python: keystoneauth re-validates and renews the token on each
call, returns `None` for missing values, and wraps client failures in
`SDKException`; Go returns empty strings or typed errors that wrap the native
cause. Catalog entries are typed `CatalogService` values rather than raw
dictionaries, and only the four documented endpoint fields are kept.

## Python-only utilities without a Go counterpart

- `pprint` / `pformat` delegate to the Python standard library; Go callers use
  `fmt` or `encoding/json`.
- `FileSegment` is an internal file-like upload helper; Go callers can use
  `io.SectionReader`, and segment uploads use internal readers.
- `range_search` and `search_resources` operate on untyped dictionaries and
  string-registered resource names; the Go API uses typed collections and
  concrete filter options.
- `session` is the keystoneauth session; Go does not expose a session object.
- `close` releases a requests session and thread pool; Go owns neither (the
  caller owns the `http.Client`).
