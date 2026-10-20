# Python V1 Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v1/create_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/delete_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/download_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/find_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/get_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/images`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/update_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/update_image_properties`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/upload_image`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/wait_for_delete`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:image/v1/wait_for_status`

Status: `unsupported`.

Unresolved boundaries:

- Glance removed the v1 API and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

## References

- [image v1 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py)
- [image README](README.md)
- [`create_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L63)
- [`upload_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L254)
- [`delete_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L328)
- [`find_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L359)
- [`get_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L378)
- [`images`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L390)
- [`update_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L400)
- [`download_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L414)
- [`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L477)
- [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L515)
- [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L554)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/image.py)
