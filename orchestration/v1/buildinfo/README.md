# Heat build info

`buildinfo.API.Get` wraps Gophercloud v2.15.0 `orchestration/v1/buildinfo`.

It sends `GET /build_info` and accepts 200 only. The result is the typed `BuildInfo` with the API and engine revision strings. Errors (including 403) are returned as `gophercloud.ErrUnexpectedResponseCode`. The call takes no options. Cancelling the context stops the local call; a pre-canceled context sends no request. Python openstacksdk equivalence is tracked separately.
