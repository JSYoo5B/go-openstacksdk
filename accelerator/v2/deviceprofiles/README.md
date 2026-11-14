# Deviceprofiles

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
profile = conn.accelerator.create_device_profile(
    name="fpga-profile",
    groups=[{"resources:CUSTOM_FPGA": "1", "trait:CUSTOM_ACCEL": "required"}],
)
conn.accelerator.delete_device_profile(profile.uuid, ignore_missing=False)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [openstacksdk pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned DeviceProfile resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/device_profile.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/device_profiles.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
