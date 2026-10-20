# Heat API versions

`apiversions.API.ListVersions` wraps Gophercloud v2.15.0 `orchestration/v1/apiversions`.

It sends `GET /` against the endpoint with its version and project suffix removed (for a client endpoint `/heat/v1/{project}` the request path is `/heat/`). It is a single-page listing: no paging is followed. 200 and 204 are accepted, and 204 yields no versions. Other statuses, including 404, are returned as `gophercloud.ErrUnexpectedResponseCode`.

The call takes no options and has no extension inputs. Cancelling the context before the call makes no request. Python openstacksdk equivalence is tracked separately.
