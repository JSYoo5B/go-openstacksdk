# Python `availability_zones` route

Pinned Python `openstack/shared_file_system/v2/availability_zone.py` lists `GET /availability-zones` with `_max_microversion = "2.7"`, keeps `created_at`/`updated_at` as strings, and `cloud.list_share_availability_zones` simply materializes that list. Gophercloud's `List` reads the legacy `GET /os-availability-zone` route and decodes timestamps into `time.Time`.

`API.ListZones` / `API.AllZones` (new, SDK-owned, in `python_list.go`) reproduce the Python request: `GET /availability-zones`, success 200 only, with `X-OpenStack-Manila-API-Version: 2.7` always sent on that request (the configured client `Microversion` is not modified and not used). Rows are `Zone` values with string timestamps and the raw `Body`, `Header` and `StatusCode`; `availability_zones_links` next links are followed if a deployment emits them. `API.List` is unchanged and keeps the legacy route.

Differences from Python: no Resource object or lazy generator state beyond `iter.Seq2`; the microversion is pinned to 2.7 rather than negotiated, so a server that does not support 2.7 fails with the server's error instead of being negotiated downward. Server behavior for either route has not been verified against a deployment; the claim is that the Go request equals Python's request.
