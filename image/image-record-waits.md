# Image Record Waits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
seed = conn.image.get_image("image-id")

def show_progress(progress):
    print("progress:", progress)

ready = conn.image.wait_for_status(
    seed, "active", failures=["ERROR"], interval=2, wait=None,
    attribute="status", callback=show_progress,
)
print(ready.to_dict())

last = conn.image.wait_for_delete(
    ready, interval=2, wait=120, callback=show_progress,
)
print(last.to_dict())
```

See `Example_imageRecordWaits` in [example_image_record_waits_test.go](example_image_record_waits_test.go), compiled by `go test`.

## References

- [ImageRecord](image-records.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2363)
- [Resource wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2722)
- [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L99)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [Service Waits](../docs/service-waits.md)
- [Image Record Wait Test](image_record_wait_test.go)
- [Connection Image Record Waits Test](../connectiontest/connection_image_record_waits_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
