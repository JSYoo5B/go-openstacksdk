// Package v2 is the Glance image v2 service. Connection.ImageV2 returns a
// Service whose fields wrap Gophercloud's Glance calls and add SDK-owned task,
// staging, import, service information and identity calls; the image package
// adds the higher-level image.Service workflows.
//
// Images.WaitForState fails on the exact ERROR or error status with no SDK
// timeout, and Images.WaitForDelete defaults to 120 seconds; both poll every
// two seconds unless caller resource wait options say otherwise.
//
// `Images.TaskRecords/AllTaskRecords` provides Source task listing scoped to an
// image ID, local filters and owned options.
//
// `ServiceInfo.ListStores/AllStores(ctx, options...)` lists stores, optionally
// requesting details with `ListStoresOpts.Details = true`. `GetImportInfo(ctx)`
// reads available import methods and preserves original fields, actual headers
// and status. Discovery is current server information, not import execution or
// an automatic capability gate. Native `ImageImport.Get` remains available.
// `GetUsageInfo(ctx)` fetches authenticated-project usage/limits once; usage
// covers dynamic resources, exact int64, omitted/null and empty results when
// server reporting is disabled.
//
// `ImageData.StageImage(ctx, ref, data, options...)` verifies fresh queued
// state, sends the caller's Reader once and fetches the image again.
// `StageKnownImage` uses an existing Image ID/status snapshot to skip the
// initial GET. Optional Size is an image-size header, not inferred HTTP
// Content-Length. Actual PUT204 acceptance is separate from the later GET200,
// and acceptance evidence survives fetch failure. Native Stage/Upload/Download
// remain available.
//
// `Tasks.WaitForTask(ctx, resource.ID(id), options...)` defaults to
// success/failure, a 120-second timeout and 2-second interval. It recreates
// only when a selected failure state and the exact 396 message are observed,
// using actual fetched type/input; the new ID shares the same context deadline.
// Native Task Get/Create/List and common waits remain available.
//
// `ImageImport.ImportImage(ctx, ref, options...)` fetches and validates
// existing image formats before submitting default glance-direct import.
// `ImportKnownImage` uses an existing native Image ID/format. Concrete options
// support web/remote imports, root store lists and omitted versus false values,
// returning the actual 202 response. Native Create/Get contracts remain
// available.
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
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v2
