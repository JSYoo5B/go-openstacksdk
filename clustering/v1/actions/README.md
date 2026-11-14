# Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
action = conn.clustering.get_action(action_id)
actions = conn.clustering.actions(target_id=node_id, limit=20)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

```python
action = conn.clustering.update_action("ACTION_ID", status="CANCELLED")
```

See `Example_actionsFragment2` in [example_actions_fragment2_test.go](example_actions_fragment2_test.go), compiled by `go test`.

See `Example_actionsFragment3` in [example_actions_fragment3_test.go](example_actions_fragment3_test.go), compiled by `go test`.

See `example_actions1_test.go` in [example_actions1_test.go](example_actions1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#actions)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
- [tracked handle](../tracking/async/README.md)
