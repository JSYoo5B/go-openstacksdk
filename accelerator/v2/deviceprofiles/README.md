# Deviceprofiles

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
profile = conn.accelerator.create_device_profile(
    name="fpga-profile",
    groups=[{"resources:CUSTOM_FPGA": "1", "trait:CUSTOM_ACCEL": "required"}],
)
conn.accelerator.delete_device_profile(profile.uuid, ignore_missing=False)
```

```go
func createAndDelete(ctx context.Context, client *gophercloud.ServiceClient) error {
    profiles := deviceprofiles.New(client)
    profile, err := profiles.Create(ctx, deviceprofiles.CreateOpts{
        Name: "fpga-profile",
        Groups: []deviceprofiles.Group{{
            "resources:CUSTOM_FPGA": "1",
            "trait:CUSTOM_ACCEL": "required",
        }},
    })
    if err != nil { return err }
    if err := profiles.Delete(ctx, resource.ID(profile.UUID), resource.WithMissingError()); err != nil {
        return err
    }
    return profiles.WaitDeleted(ctx, resource.ID(profile.UUID),
        resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second))
}
```

## References

- [openstacksdk pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned DeviceProfile resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/device_profile.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/device_profiles.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
