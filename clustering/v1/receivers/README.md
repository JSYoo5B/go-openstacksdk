# Receivers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
receiver = conn.clustering.create_receiver(
    name="scale_out", type="webhook", cluster_id="CLUSTER_ID",
    action="CLUSTER_SCALE_OUT", params={},
)
message = conn.clustering.create_receiver(name="events", type="message")
receiver = conn.clustering.get_receiver(receiver.id)
```

See `Example_receiversFragment1` in [example_receivers_fragment1_test.go](example_receivers_fragment1_test.go), compiled by `go test`.

```python
receiver = conn.clustering.update_receiver(
    "scale_out", name="scale_in", action="CLUSTER_SCALE_IN", params={},
)
```

See `Example_receiversFragment2` in [example_receivers_fragment2_test.go](example_receivers_fragment2_test.go), compiled by `go test`.

```python
receivers = conn.clustering.receivers(type="message", user_id="USER_ID", limit=20)
```

See `Example_receiversFragment3` in [example_receivers_fragment3_test.go](example_receivers_fragment3_test.go), compiled by `go test`.

```python
receiver = conn.clustering.find_receiver("scale_out", ignore_missing=True)
conn.clustering.delete_receiver("scale_out", ignore_missing=True)
```

See `Example_receiversFragment4` in [example_receivers_fragment4_test.go](example_receivers_fragment4_test.go), compiled by `go test`.

See `Example_receiversFragment5` in [example_receivers_fragment5_test.go](example_receivers_fragment5_test.go), compiled by `go test`.

See `example_receivers1_test.go` in [example_receivers1_test.go](example_receivers1_test.go), compiled by `go test`.

## References

- [Receivers](../tracking/receivers/README.md)
- [Finding](../finding/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#receivers-receivers)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
