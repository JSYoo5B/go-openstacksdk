# Deployables

`deployables.API` lists and reads Cyborg deployables (logical acceleration units). The SDK owns this model because pinned Gophercloud has no accelerator service.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
values, err := accel.Deployables.All(ctx, resource.WithQuery("name", "compute-1_FPGA"))
```

Behavior:

- `All` and `List` send `GET /v2/deployables`. Without filters no query string is sent; `resource.WithQuery` forwards explicit values unchanged.
- Caller-supplied `limit` and `marker` queries are rejected before any HTTP request. Only server continuation links (`next`) are followed.
- An empty result is a non-nil empty slice. A native HTTP failure (including 404) stays a `gophercloud.ErrUnexpectedResponseCode` cause.
- A pre-canceled context sends no request; an expired deadline keeps `context.DeadlineExceeded` as the cause.
- Null identifiers such as `parent_id` decode as nil pointers rather than zero values.

Evidence: `api/cyborg_cloud_lists_test.go` (HTTP fixtures only; no live Cyborg deployment).
