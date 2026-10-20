# Manila services

`services.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/services` and exposes one read-only administrative operation, `List`, for `GET /services`. Policy normally restricts it to administrators; a denial surfaces as an HTTP response error. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types.

The response is a single page of `Service` (`ID`, `Binary`, `Host`, `Zone`, `State`, `Status`, `UpdatedAt`); `updated_at` null decodes to the zero time and `next` links are not followed. A 204 response yields no items.

Known upstream limitation: the pinned Gophercloud `ListOpts` fields are tagged for JSON rather than query encoding, so `WithListOptions` filters such as `Host` or `State` are not sent. Use `WithListQuery("host", value)` (or the other query names `binary`, `zone`, `state`, `status`) to filter on the server. Nil options fail with `resource.ErrInvalidOption` before any request.

There is no `Collection`, get, enable or disable operation in this package. Cancelling a context stops the local call.
