# Compute service lookup

`FindRecord` follows the pinned `compute.find_service` override. It sends only LIST, compares Python-style string IDs or the `name` descriptor (which falls back to `binary`), then applies the optional `host` filter locally. Source discards other query kwargs rather than sending them to Nova. Duplicate matches are errors; missing defaults to nil, or errors with `request.IgnoreMissing(false)`.

`ReadRecord.Resource` contains declared descriptors, `Wire` retains response extensions, and `Envelope`, `Header`, `StatusCode` retain actual HTTP evidence. Version discovery chooses at most 2.69 when the client has no selected version. Typed headers/microversion are explicit Go transport conveniences; Source's query kwargs do not gain those meanings.

```python
import openstack
conn = openstack.connect()
service = conn.compute.find_service("nova-compute", host="node", ignore_missing=False)
print(service.id)
```

See `Example_findRecord` in [example_find_record_test.go](example_find_record_test.go), compiled by `go test`.

Go accepts explicit string identities and finite JSON options rather than Python Resource instances and arbitrary object equality. `RecordOptions.ListBasePath` selects a service-relative list path; route validation rejects external origins and traversal. This mapping leaves shared Resource/session/live-cloud parity contracts open.

A failure while preparing version discovery returns an error before a resource record exists. `Discovery` evidence is attached to successful resource records; this binding does not return a discovery-only receipt when preparation fails. The error retains the underlying failure, and this boundary is not live-cloud validation.
