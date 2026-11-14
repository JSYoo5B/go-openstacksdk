# Python List Query

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for mirror, err := range api.List(ctx,
	request.Query("name", "mirror"),
	request.Query("fields", []string{"id", "name"}),
	request.Query("sort_key", []string{"name", "id"}),
	request.Query("sort_dir", []string{"asc", "desc"}),
) {
	...
}
```

## References

- [`tap_mirrors(**query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`TapMirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_mirror.py)
