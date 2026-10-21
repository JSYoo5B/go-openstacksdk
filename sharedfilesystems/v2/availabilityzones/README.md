# Manila availability zones

`availabilityzones.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/availabilityzones` and exposes one read operation, `List`, for `GET /os-availability-zone`. It takes no filters and has no extension options. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types; this package does not negotiate it.

The response is a single page of `AvailabilityZone` (`ID`, `Name`, `CreatedAt`, `UpdatedAt`). Timestamps use the Manila no-zone format; a missing or null timestamp decodes to the zero time. Links are not followed. Non-200 statuses surface as HTTP response errors, and a cancelled context fails before any request.

Python parity for the `/availability-zones` route (`ListZones`, `AllZones`): see [python-list.md](python-list.md).
