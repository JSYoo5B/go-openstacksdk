// Package image exposes Glance images through shared resource policies and a
// metadata creation, direct upload, and optional active-wait workflow.
// Connection.Image returns its Service.
//
// `image.Service.UploadImage` uses concrete `ImageUploadOpts` and raw metadata
// fields for image POST201 followed by binary PUT204. It preserves the creation
// ImageInfo and both stages' actual bodies, headers and statuses; the caller's
// borrowed reader is sent once. Creation information survives a later error.
//
// `image.Service.UpdateImage/SetImageProperties` provides ordered PATCH and
// add-upsert through concrete `ImagePatch` and property options. The SDK
// escapes JSON Pointer keys, submits
// application/openstack-images-v2.1-json-patch once and returns actual200
// `ImageInfo`. Empty changes submit [] and observe metadata.
//
// `image.Service.GetImage/ListImages/AllImages` uses concrete options and
// SDK-owned `ImageInfo` for fetch/list. It resolves names or fetches IDs and
// preserves nullable canonical fields, exact numbers and raw JSON extension
// properties. Listing validates canonical body next links and retains repeated
// filters. MaxItems and SinglePage are local consumption policies.
//
// `image.Service.ImageTasks/AllImageTasks` requires an image Ref and concrete
// header/MaxItems options. IDs are used directly and names resolve exactly. One
// GET returns a finite array of SDK-owned `ImageTaskInfo`, preserving nullable
// deleted/deleted_at and raw JSON; MaxItems is local consumption.
//
// `image.Service.CreateTask/GetTask/Tasks/AllTasks` provides general Task
// creation, fetch and listing using concrete options and SDK-owned `TaskInfo`.
// Input defaults to {}; the type query reaches the request. Lists follow
// canonical body next links; MaxItems/SinglePage are local. Native Task APIs
// and the 396 wait workflow remain available.
//
// `image.Service.AddImageMember/GetImageMember/UpdateImageMember/RemoveImageMember/FindImageMember/ListImageMembers/AllImageMembers`
// requires a parent image Ref, literal member ID and concrete
// header/missing/MaxItems options. Find/Remove default nil IgnoreMissing to
// true and return nil only after owned GET404/DELETE404 responses are handled
// successfully. Find fetches an ID directly, without Python's 400/403/list
// fallback. List is a lazy finite array; MaxItems is local. Native Member APIs
// and Members.InImage remain available; member Name/project lookup and
// automatic query/pagination are not synthesized.
//
// `image.Service.GetImageCache/QueueImage/CacheDeleteImage/ClearCache/CachedImageNodes/CleanCache/PruneCache`
// uses fixed cache paths and concrete options. Delete defaults nil
// IgnoreMissing to true, ignoring only an actual DELETE404 whose body is fully
// read and closed. Clear's zero target means both cache and queue and omits the
// header. Clean returns an actual200 opaque response; Prune returns actual200
// counts. No discovery, active prefetch, automatic fanout or completion
// inference is added.
//
// `image.Service.GetSchemaRecord` selects the same 16 fixed paths with
// `SchemaKind`, applying ordinary Schema and MetadefSchema nullable
// raw/dict/bool/list conversion. It separates Connection-location-enriched
// `Resource` from actual `Wire`, `Envelope` and response evidence.
// Empty/non-JSON HTTP200..399 responses retain bare defaults.
//
// `image.Service.AddImageLocation` submits a URL and optional validation hash
// through a fixed POST, retaining actual202 acknowledgement.
// `GetImageLocations` performs one body/query-free GET200 of a finite array and
// returns URLs, metadata and original response. Native location-set PATCH
// remains available; location IDs, pagination and completed hash verification
// are not synthesized.
//
// `image.Service.AddTag/RemoveTag` changes one tag through fixed PUT/DELETE
// paths; `DeactivateImage/ReactivateImage` submits a bodyless POST action. IDs
// bypass lookup; names resolve exactly. Missing resources are errors. Original
// actual204 acknowledgement, headers and status are retained. Native
// `ReplaceImageTags` and metadata Update remain available; these actions do not
// automatically fetch or wait for state.
//
// `image.Service.DeleteImage(ctx, ref, options...)` deletes an image or the
// store copy selected by `DeleteImageOpts.StoreID`. IDs issue a fixed DELETE
// directly; names resolve exactly; missing resources are ignored by default. It
// retains original actual204 response, headers and status without hiding
// read/close/cancellation errors as absence. Native `Images.Delete` retains its
// 202/204 contract and resource inventory.
//
// `image.Service.DownloadTo(ctx, ref, writer, options...)` fetches fresh
// metadata, downloads to the writer in selected store order and checks
// available checksums. It does not close the caller's writer. It preserves
// actual metadata/binary responses, bytes written and integrity results. Native
// `ImageData.Download` retains its stream-result contract.
//
// `image.Service.CreateAndImport` validates concrete metadata/stage/import/wait
// policy before HTTP, then connects create201, one staging204, fresh fetch200
// and import acceptance202. Remote import skips staging; active-state waiting
// is explicit. This workflow belongs to the higher-level service from
// `conn.Image(ctx)` rather than additional versioned native operations.
//
// The 16 `image.Service.Get*Schema` methods issue body/query-free GETs to fixed
// schema paths. Concrete `GetSchemaOption` and SDK-owned `image.Schema`
// preserve canonical fields, omitted/null values and exact raw JSON, including
// bool/object additionalProperties. Names, links and URLs remain passive data,
// without ID lookup, pagination, automatic validation or Resource cache.
package image
