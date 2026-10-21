# Limits

SDK-owned `GET /limits` (absent from Gophercloud v2.15.0). `Get(ctx, ...)` sends one request (only 200 succeeds) and returns `Limits` with typed `Absolute` counters (`*int64`, null or absent is nil), raw `Rate` rows and `Evidence` (actual `limits` object, headers, status). No microversion floor is applied. Query extensions go through `WithQuery`.

Python comparison: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
