# Metric (Prometheus-compatible) API

`metrics.API` wraps Gophercloud v2.15.0 `metric/v1/metrics`, which binds the Aetos Prometheus-compatible HTTP API. It is not the Gnocchi API used by Python openstacksdk's metric service; there is no openstacksdk Resource equivalent.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Query` | `GET /query` | 200 | `QueryOpts.Query` is required; `Time` and `Timeout` are sent only when non-empty |
| `Labels` | `GET /labels` | 200 | `Match` is sent as repeated `match[]`; `Start`, `End` optional |
| `LabelValues` | `GET /label/{name}/values` | 200 | Same optional filters as `Labels` |
| `Series` | `GET /series` | 200 | `Match` (at least one `match[]`) is required |
| `Targets` | `GET /targets` | 200 | Optional `State` |
| `RuntimeInfo` | `GET /status/runtimeinfo` | 200 | Typed `RuntimeInfoData`; false and zero values are decoded as given |
| `Snapshot` | `POST /admin/tsdb/snapshot` | 200 only | No request body; returns the snapshot name |
| `CleanTombstones` | `POST /admin/tsdb/clean_tombstones` | 204 only | No request body |
| `DeleteSeries` | `POST /admin/tsdb/delete_series` | 204 only | `Match` is required; matchers travel in the query string, there is no body |

Responses use the Prometheus envelope. An HTTP 200 whose `status` is `error` returns an error built from `errorType` and `error`; other statuses stay `gophercloud.ErrUnexpectedResponseCode` (including 403 for the admin routes when the caller lacks permission). Extension queries use `With...Query` options (for example `WithQueryQuery`, `WithDeleteSeriesQuery`) and replace a typed value of the same name; field and header extensions do not exist. Missing required inputs and nil options fail with a local error (`resource.ErrInvalidOption` for options) before any request. `DeleteSeries` without options fails locally; it never deletes everything.

`Snapshot`, `CleanTombstones` and `DeleteSeries` change server state, are never replayed automatically, and a deadline or cancellation after sending leaves the remote outcome unknown. No microversion header is involved. The common `resource.Collection` is not used.
