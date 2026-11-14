# Keystone Policy list records

`service.Policies.ListRecords` and `AllRecords` provide the usable Go alternative for pinned openstacksdk `policies(**query)`. Policy declares only `limit` and `marker` as server query parameters. The SDK applies canonical Body filters (`id`, `name`, `blob`, `links`, `project_id`, `type`, `user_id`) locally after loading each Resource. Nested object filters use recursive subset matching. Unknown filter keys are ignored before JSON encoding.

Each `PolicyRecord` owns a nullable, untyped declared `Resource` view, complete remote `Wire` fields, whole page `Envelope`, `Header` and `StatusCode`. Missing fields default to null; unknown response extensions remain in Wire. Return IDs and links are passive data. `request.Limit(0)` preserves explicit zero, while an omitted limit is absent. MaxItems counts raw rows before filtering, Paginated defaults true, and AllRecords preserves already consumed results on a later page/row error.

## Standalone Python

```python
import openstack


def main():
    conn = openstack.connect()
    for policy in conn.identity.policies(
        project_id="project-id", type="application/json", limit=100
    ):
        print(policy.id, policy.blob)


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_IDENTITY_ENDPOINT` to the Keystone v3 service endpoint and `OS_TOKEN` to an authorized token.

See `Example_listRecords` in [example_list_records_test.go](example_list_records_test.go), compiled by `go test`.

## Options and Go boundaries

`WithRecordListOpts` snapshots the complete concrete policy. Individual helpers select Limit, Marker, MaxItems, Paginated, canonical Body filters and headers; nil options or negative limits/caps fail before HTTP. Iterator creation is lazy and reusable. Changing the selected API or route during callbacks fails without sending. Physical/context/source failures and malformed rows are errors with actual response evidence; they cannot become an empty successful list.

Existing generated List/All retain their native behavior; use ListRecords for correct Policy Body filtering. This concrete profile does not expose arbitrary Proxy kwargs, `__conflicting_attrs`, custom base paths, JMESPath expressions, or microversion overrides. Safe fixed paths, finite owned JSON, cycle checks, immutable records and Go error iteration are deliberate Go adaptations. Shared Resource/Connection/cache/runtime behavior, real Keystone authorization and release validation remain SDK-wide gaps. This operation mapping does not assert complete openstacksdk parity.
