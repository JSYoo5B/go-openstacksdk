# Devices

`devices.API` lists and reads Cyborg physical accelerator devices. The SDK owns this model because pinned Gophercloud has no accelerator service.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
values, err := accel.Devices.All(ctx, resource.WithQuery("hostname", "compute-1"))
```

Behavior:

- `All` and `List` send `GET /v2/devices`. Without filters no query string is sent; `resource.WithQuery` forwards explicit values unchanged.
- Caller-supplied `limit` and `marker` queries are rejected before any HTTP request. Only server continuation links (`next`) are followed, so a paged result is complete.
- An empty result is a non-nil empty slice. A native HTTP failure stays a `gophercloud.ErrUnexpectedResponseCode` cause.
- A pre-canceled context sends no request; an expired deadline keeps `context.DeadlineExceeded` as the cause.

Evidence: `api/cyborg_cloud_lists_test.go` (HTTP fixtures only; no live Cyborg deployment).
