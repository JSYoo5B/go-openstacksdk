# Servers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
encrypted, err := service.Servers.GetPassword(ctx, serverID)
if err != nil { return err }
_ = encrypted

password, err := service.Servers.GetPassword(ctx, serverID,
    servers.WithGetPasswordPrivateKey(privateKey))
if err != nil { return err }
_ = password
```

## References

- [User Read APIs](../../user-read-apis.md)
- [Keypairs Console](../../keypairs-console.md)
- [Python Server.get_password](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)
- [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Service Waits](../../../docs/service-waits.md)
