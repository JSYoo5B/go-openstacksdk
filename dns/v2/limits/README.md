# Limits

SDK-owned binding for `GET /v2/limits` (200 only), reached with `service.Limits()`. The flat object is decoded into `Limits` with `Body`, `Header` and `StatusCode` retained, so null and unknown fields stay visible. Python `limits()` is a generator that yields this single object; Go returns it directly.
