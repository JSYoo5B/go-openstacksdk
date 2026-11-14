# Python Attachments Groups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
for attachment in conn.compute.volume_attachments(
    server_id, limit=20, offset=0, max_items=100,
):
    print(attachment.to_dict())

conn.compute.update_volume_attachment(
    server_id, current_volume_id, delete_on_termination=False,
)
conn.compute.update_volume_attachment(
    server_id, current_volume_id, volume_id=replacement_volume_id,
)
group = conn.compute.find_server_group(group_name, ignore_missing=False)
```

See `Example_attachmentsGroups` in [example_attachments_groups_test.go](example_attachments_groups_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
