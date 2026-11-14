// Package images exposes Gophercloud v2.15.0's openstack/image/v2/images calls
// through API, with concrete options, the SDK's request call categories and
// typed results. Connection.ImageV2 returns the image/v2 service, whose Images
// field provides this API.
//
// `Images.FindIdentity(ctx, identity, options...)` resolves name/ID strings
// automatically, with list fallback after GET400/403/404, exact ID/name and
// duplicate checks, and a default missing result of `nil, nil`. Select strict
// handling with `resource.WithIdentityFindIgnoreMissing(false)`. Caller
// `WithIdentityFindQuery` filters survive GET and fallback listing; automatic
// name hints apply only to listing. Concrete options control query/fallback.
//
// Glance identity finding searches hidden images once after a successful
// ordinary inventory has no match, applying `os_hidden=true` to the original
// caller query. The second list gets no automatic name hint or extra GET.
// Errors, duplicates and cancellation from the first search return immediately;
// final missing policy applies only after both lists succeed without a match.
// This maps Python `find_image(name_or_id, ignore_missing=True)`; raw wire
// queries are a Go extension.
package images
