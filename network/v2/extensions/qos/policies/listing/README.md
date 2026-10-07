# QoS Policy list filters

`Resources.List` and `Resources.All` classify the pinned OpenStackSDK attributes into server query parameters and local response conditions. Native `API.List`, GET, explicit reference lookup, and `API.FindIdentity` retain their existing contracts.

For non-shared policies with an empty rule array, use the semantic fields and a local raw-row cap:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    qospolicies "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listPoliciesWithoutRules(ctx context.Context, client *gophercloud.ServiceClient) ([]*qospolicies.Policy, error) {
    return qospolicies.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "is_shared": false,
            "fields": []string{"id", "name", "rules", "shared"},
            "rules": []any{},
        }),
        resource.WithMaxItems(20),
    )
}
```

```python
policies = list(conn.network.qos_policies(
    is_shared=False,
    fields=["id", "name", "rules", "shared"],
    rules=[],
    max_items=20,
))
```

The deprecated tenant field is a local condition, distinct from a server project query. An explicit Body option can compare an expected complete rule array without constructing an upstream builder or predicate:

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/gophercloud/gophercloud/v2"
    qospolicies "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listTenantPolicies(ctx context.Context, client *gophercloud.ServiceClient, expectedRules json.RawMessage) ([]*qospolicies.Policy, error) {
    return qospolicies.New(client).Resources.All(ctx,
        resource.WithFilter("tenant_id", "legacy-tenant"),
        resource.WithBodyFilter("rules", expectedRules),
        resource.WithPaginated(false),
    )
}
```

```python
def list_tenant_policies(conn, expected_rules):
    return list(conn.network.qos_policies(
        tenant_id="legacy-tenant",
        rules=expected_rules,
        paginated=False,
    ))
```

`expectedRules` is the complete JSON array to compare, including every key in every rule object. Objects inside a rule array are compared exactly; partial rule dictionaries are not subset predicates. When precise large numbers matter, supply their original JSON values rather than numbers already converted through a native `map[string]any` model. Requested `fields` should include the response fields needed by local conditions.

| Canonical server query | Wire spelling |
| --- | --- |
| `description`, `fields`, `id`, `is_default`, `limit`, `marker`, `name`, `project_id`, `sort_dir`, `sort_key`, `tags` | Same as the attribute |
| `is_shared` | `shared` |
| `any_tags` | `tags-any` |
| `not_tags` | `not-tags` |
| `not_any_tags` | `not-tags-any` |

These are 15 canonical query fields and 19 accepted spellings. Both attribute and wire spelling are accepted. In one bulk map canonical presence wins over its wire alias, including null, false, an empty string, or an empty array. Individual options for the same wire target use the last selected value. Query fields remain server-only: semantic `id`, `name`, tags, default, and shared do not activate local predicates on returned rows. Existing `resource.WithName` keeps its exact local name comparison and server hint; it conflicts with semantic `name`.

The only local Body fields, through either semantic or explicit Body options, are `rules` and `tenant_id`. QoSPolicy inherits Python `Resource` plus `TagMixin`, rather than `NetworkResource`. Native fields `created_at`, `updated_at`, and `revision_number` are not declared Python filter attributes; semantic options discard these names. Tags are queried rather than locally compared. `tenant_id` reads its own original field and does not fall back to `project_id`, even though the Python project descriptor has a deprecated tenant alias.

Local conditions compare original JSON rows paired with the native typed values. Rule arrays compare their entire ordered values, including null map elements and full nested objects. Missing and null whole arrays match a null filter, while an empty array is distinct. Native null map elements and inner null values were already preserved and retain that behavior. Missing/null `tenant_id` match null rather than the native zero string `""`; a present empty string remains distinct.

After successful native decoding, raw rule numbers retain exact decimal equality: `1`, `1.0`, and `1e0` compare equal, while `9007199254740993` remains different from `9007199254740992`. This improves the earlier selector's native float64 projection. Returned `Policy.Rules` still uses Gophercloud's `[]map[string]any`, so its numbers may already be rounded to float64; numbers outside the native float64 range can fail decoding before raw matching. Raw fields are used for comparison without replacing the returned native model or adding response metadata. Go distinguishes booleans from numbers and strings from numbers; Python's boolean/number equality is intentionally different. A dictionary filter against a non-object returns nonmatch rather than Python's possible attribute error.

The native whole-page decoder remains authoritative before raw selection. Missing/null string and boolean fields retain native zero values; incompatible non-null values fail decoding. `TenantID` is a native string, so an arbitrary numeric/object tenant response fails before the raw condition. Rules must decode as an array of objects/nulls; Python's list descriptor may wrap a non-list value, which this layer does not imitate. Shared/default booleans, integer revision, string tags, and RFC3339 timestamps retain their native schema. The native Policy has no custom timestamp decoder for the older format without a timezone.

Options own JSON snapshots at construction and application. Individual semantic options merge and the final selected value wins. Bulk `WithFilters` replaces only its namespace; nil or an empty map clears it. Only final selected semantic values are validated, so replacement/clear can remove an earlier captured encoding error. Explicit Body and raw query options retain their separate snapshots, namespace, and error behavior. Unsupported bindings reject even an explicit semantic clear.

Known query values accept strings, booleans, JSON numbers, or a scalar array encoded as repeated values. Booleans use lowercase text and number spelling is retained. Null array elements are omitted; null values and empty arrays omit URL values while retaining key presence for precedence and collision checks. Maps and nested arrays are invalid query values. Unknown semantic fields are discarded along with their encoding errors. The reserved source controls `max_items`, `paginated`, `base_path`, `allow_unknown_params`, `session`, `headers`, `microversion`, `resource_type`, and `jmespath_filters` require dedicated supported controls rather than field options.

Semantic query and raw query cannot select the same wire key, even with equal values or a null/empty semantic value. Semantic `limit` conflicts with `WithPageSize`, and semantic `name` conflicts with `WithName`. A semantic local field conflicts with an explicit Body condition for the same field. Local Body and raw query with the same text remain independent: `WithFilter("rules", ...)` compares rows, while `WithQuery("rules", ...)` is an explicit wire extension. Raw query spellings are not automatically transposed; existing native typed List and FindIdentity retain their own query behavior, including raw status. `WithStatus` remains unsupported because the native Policy has no status field.

The list retains the Gophercloud v2.15.0 Policy pager, native statuses 200/204/300, and `policies_links` continuation. The native helper selects the last matching `rel: next` href. Top-level `next`, `links.next`, and HTTP Link headers do not become new native continuation paths. Native JSON parsing precedes page emptiness checks, so an empty 204 with JSON Content-Type may fail EOF. The native iterator may follow a foreign advertised next URL; this unit introduces no same-origin guard, fixed query continuation policy, or synthetic marker fallback. Shared cycle detection, later HTTP/decode errors, and cancellation remain terminal.

`WithMaxItems` counts raw rows before local Body/name filters, does not refill the cap with matching rows, and does not invent a wire limit. Python also counts raw rows; its limit hint and continuation timing differ from the immediate Go cap. Whole-page native decoding can fail on malformed rows beyond the cap. A null row consumed by the Body-filter iterator is invalid, while the query-only ordinary iterator still retains the native zero-value row. An empty object is valid. A cap or consumer break can avoid consuming later raw null rows, but cannot bypass native whole-page errors. `WithPaginated(false)` stops after the first page, and `All` returns nil on a terminal error rather than partial rows.

This ordinary list layer does not implement the full Python Resource default/alias/unknown-Body cache or dirty lifecycle, descriptor coercion, arbitrary per-call base path/header/microversion/session, deprecated JMESPath expressions, or every inherited pagination path. QoSPolicy has no class-specific maximum microversion negotiation requirement. Configure the source client before concurrent calls; the cached Connection collection retains its shared provider, live token, transport, configured headers, context, and ResourceBase. See [shared listing controls](../../../../../../docs/listing.md).

Source: pinned OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, [QoSPolicy declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21-L66), [TagMixin query aliases](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L28-L38), [declared proxy list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5177-L5193), and [Resource query/filter handling](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217-L2358). Native extraction and page behavior come from [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/qos/policies/results.go).

HTTP evidence is in the [seven API groups](../../../../../../api/qos_policy_list_filters_test.go) and [two cached Connection groups](../../../../../../connection_qos_policy_filters_test.go). These are mock HTTP contract checks rather than live-cloud tests.
