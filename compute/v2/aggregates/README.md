# Compute aggregates

`compute.Service.Aggregates` exposes aggregate CRUD and actions. SDK-owned additions include [UpdateRecord](python-update-record.md) and [image precaching](python-precache-images.md). PrecacheImages accepts a slice of literal image IDs, owns microversion2.81, and returns an error-only acknowledgement without decoding an action Resource. See the precaching guide for standalone Python/Go examples and documented Go representation differences.
