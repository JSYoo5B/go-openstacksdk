# AddressGroup list filters

`Resources.List` and `Resources.All` classify the pinned OpenStackSDK list attributes as server query parameters or local response conditions. The generated `API.List`, explicit reference lookup, and `API.FindIdentity` keep their existing native contracts.

For a server query plus an exact local address list:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listAddressGroups(ctx context.Context, client *gophercloud.ServiceClient) ([]*addressgroups.AddressGroup, error) {
    return addressgroups.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "project_id": "project-alpha",
            "name": "server-name-filter",
            "addresses": []string{"192.0.2.1", "198.51.100.0/24"},
        }),
        resource.WithMaxItems(20),
    )
}
```

The corresponding Python call is:

```python
groups = list(conn.network.address_groups(
    project_id="project-alpha",
    name="server-name-filter",
    addresses=["192.0.2.1", "198.51.100.0/24"],
    max_items=20,
))
```

To keep the deprecated tenant attribute distinct from the server project query, use its local condition. Explicit `WithBodyFilter` is also available for the three audited local fields:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listTenantAddressGroups(ctx context.Context, client *gophercloud.ServiceClient) ([]*addressgroups.AddressGroup, error) {
    return addressgroups.New(client).Resources.All(ctx,
        resource.WithFilter("tenant_id", "legacy-tenant"),
        resource.WithBodyFilter("addresses", []string{}),
        resource.WithPaginated(false),
    )
}
```

```python
groups = list(conn.network.address_groups(
    tenant_id="legacy-tenant",
    addresses=[],
    paginated=False,
))
```

The eight query attributes use identical attribute and wire spellings:

| Server query | Local Body condition |
| --- | --- |
| `limit`, `marker`, `fields`, `sort_key`, `sort_dir`, `name`, `description`, `project_id` | `id`, `tenant_id`, `addresses` |

Semantic `name`, `description`, and `project_id` are server-only. Returned rows are not automatically compared against these query values. Existing `resource.WithName` additionally performs an exact local name comparison and supplies its existing server hint. Combining semantic `name` with `WithName` is an invalid namespace collision. `tenant_id` never falls back to `project_id`; the typed native model exposes only `ProjectID`, while the local tenant condition reads the original `tenant_id` response field.

The local conditions use original JSON fields paired with the native typed rows. Missing fields and explicit null match a null filter; empty strings and empty arrays remain distinct. Address arrays compare every element, order, and length, without CIDR normalization. A null address element remains null for matching, even though the native `[]string` model decodes that element as `""`. This improves the earlier typed address selector, which compared the already collapsed native slice. Missing and null entire address arrays still both match null.

Raw `tenant_id` values retain exact JSON numbers and unknown objects, despite being absent from the native model. Objects use recursive subset matching, and objects inside arrays compare exactly. An empty actual object does not match an object filter, following the pinned Python dictionary-filter behavior. Go distinguishes JSON booleans from numbers; it does not use Python's `True == 1` equality or coerce filter strings to numbers. Native known fields remain authoritative: incompatible non-null values for `id`, `name`, `description`, or `project_id` fail string decoding, while missing/null values leave the native zero string. Missing/null `addresses` leave a nil slice; other values must decode as an array of strings/nulls, with null elements becoming native empty strings. Python's list descriptor may wrap a non-list response value into a list; this Go layer retains the native decoder instead.

Options own JSON snapshots. Individual semantic options merge, and the last selected value for the same field wins. Bulk `WithFilters` replaces only the semantic namespace; nil or an empty map clears it. Only final selected values are validated, so replacement/clear can remove an earlier semantic encoding error. Explicit Body options and raw query options keep their separate behavior and namespaces. Their captured errors are not cleared by `WithFilters`.

Known query values accept strings, booleans, and JSON numbers, or a scalar array encoded as repeated values. Number spelling is retained, booleans are lowercase, and null array elements are omitted. Null values and empty arrays emit no URL values while preserving key presence for collision checks. Maps and nested arrays are invalid query values. Unknown semantic fields are discarded, including their encoding errors. The reserved source controls `max_items`, `paginated`, `base_path`, `allow_unknown_params`, `session`, `headers`, `microversion`, `resource_type`, and `jmespath_filters` require supported dedicated controls rather than field options.

Semantic query and raw query options cannot select the same wire key, even with equal values or an omitted/null semantic value. `WithPageSize` also conflicts with semantic `limit`. A semantic local field conflicts with an explicit Body condition for that field. A local Body condition and a raw query with the same text are independent: `WithFilter("addresses", ...)` compares rows, while `WithQuery("addresses", ...)` remains an explicit wire extension. `WithBodyFilter` accepts only `id`, `tenant_id`, and `addresses`; query-capable fields are not expanded into additional local selectors. Raw status remains a wire extension, and `WithStatus` remains unsupported because the native AddressGroup has no status field.

The list continues to use Gophercloud v2.15.0's AddressGroup pager and extractor. It accepts native list statuses 200/204/300 and follows `address_groups_links` entries with `rel: next`; a top-level `links.next` does not become a continuation. Native JSON decoding occurs before page emptiness checks, so an empty 204 with JSON Content-Type may return EOF. Native extraction validates the entire current page before a local row cap, including malformed known fields after the cap. A null row consumed by a Body-filter iterator is invalid; the ordinary query-only iterator still preserves the native zero-value row. An empty object is valid. Caps and early break can avoid consuming later null rows, but cannot bypass native whole-page decode failures.

`WithMaxItems` counts raw rows before local Body or name comparisons and does not invent a wire limit. `WithPaginated(false)` stops after the first page; consumer break stops fetching further pages. `All` returns nil on a terminal error rather than returning partial rows. The native continuation lane retains its existing foreign-next behavior and shared cycle detection; this unit adds no same-origin guard, marker synthesis, or fixed query continuation policy. Configure the source client before concurrent calls; its provider token, transport, headers, context, and ResourceBase remain shared with the cached Connection service.

This is an ordinary list filtering layer. It does not add tracked lifecycle/cache, generic per-call base paths or microversions, JMESPath filtering, permissive unknown response attributes to the native model, or Python's exact descriptor coercion and limit-hint/continuation policy. Native `API.List`, GET, and `FindIdentity` do not acquire ambient semantic filters. See the [shared listing options](../../../../../../docs/listing.md) for common local controls.

Source: pinned OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, [AddressGroup attributes and query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/address_group.py#L22-L59), [declared proxy list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L392), and [Resource list/filter behavior](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2210-L2358). The native page and model come from [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/security/addressgroups/results.go).

HTTP evidence is in the [seven API groups](../../../../../../api/address_group_list_filters_test.go) and [two cached Connection groups](../../../../../../connection_address_group_filters_test.go). These are mock HTTP contract checks rather than live-cloud tests.
