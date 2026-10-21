# TLDs

SDK-owned binding for Designate `/v2/tlds`. Reached with `service.TLDs()` or `tlds.New(client)`. Same wire rules as [blacklists](../blacklists/README.md) (flat bodies, explicit codes 201/200/200/204, links.next paging, null/omit preservation, extension collision checks) with `name` and `description` fields and list filters `Name`, `Description`, `Limit`, `Marker`, `SortKey`, `SortDir`.

Lookup: `Find(ctx, resource.ID(..))` is one GET; `Find(ctx, resource.Name(..))` lists with `name=` and fails on duplicates. `FindIdentity(ctx, identity)` is Python `find_tld`: GET the identifier first, fall back to a name-filtered list after GET 400/403/404, match ID or name, absence returns `nil, nil` unless `resource.WithIdentityFindIgnoreMissing(false)` is passed. Python swallows every SDK exception from the GET; Go falls back only on 400/403/404.
