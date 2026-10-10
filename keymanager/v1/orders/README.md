# Barbican native orders

`orders.New(client)` exposes the pinned Gophercloud `v2.15.0`
[order requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/keymanager/v1/orders/requests.go)
unchanged. An order asks Barbican to generate a secret asynchronously; the
created order is a reference to poll, not the secret itself.

```go
api := orders.New(client)
order, err := api.Create(ctx, orders.CreateOpts{
    Type: orders.KeyOrder,
    Meta: orders.MetaOpts{Algorithm: "aes", BitLength: 256, Mode: "cbc"},
})
if err != nil { return err }
fmt.Println(order.OrderRef)
```

| Method | Request | Accepted statuses | Result |
|---|---|---|---|
| `Get(ctx, id)` | `GET orders/{id}` | 200 | `Order` |
| `List(ctx, options...)` | `GET orders` with `limit`/`offset` | native pager | lazy `Order` rows |
| `Create(ctx, opts, options...)` | `POST orders` | 202 | `Order`, usually only `order_ref` |
| `Delete(ctx, id)` | `DELETE orders/{id}` | 202, 204 | error only |

`CreateOpts.Type` and the `algorithm`, `bit_length` and `mode` meta fields have
no `omitempty`, so empty values are sent; `name` and `payload_content_type`
are omitted when empty. `Meta.Expiration` is formatted without a zone, and
response times decode with the native RFC3339-without-zone parser.
`WithCreateField` adds extension JSON beside `type` and `meta` and rejects
keys already in the body. IDs are joined into the path without escaping.

`List` serializes `ListOpts` plus `WithListQuery` values, follows the body
`next` link exactly as returned and stops on an empty page. Other statuses
keep the native `gophercloud.ErrUnexpectedResponseCode`; the SDK adds only
`resource.OperationError{Resource: "orders"}` context.
