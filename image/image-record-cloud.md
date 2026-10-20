# Image Record Cloud

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_image/ImageCloudMixin/list_images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllCloudImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilterDeleted`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryShowAll`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- AllCloudImageRecords eagerly consumes the complete default owned image list once, keeps every consumed row in Inventory and removes rows whose Python-lowered string status equals deleted unless FilterDeleted is false. Value is the completed JSON array of kept declared views, [] for an empty result, and Images aligns with it.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListDeletedFilterShowAllAndLocation`.
- ShowAll true overrides filter_deleted before the v2 query choice and adds only member_status=all; explicit false keeps the query-free default. The shared Connection location is projected into every Resource while actual Wire location is retained.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListDeletedFilterShowAllAndLocation`, `connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- With the status filter enabled a null, numeric or array status fails with the actual list-page ResponseError and retains Inventory through the failing row without completed Value/Images; disabled filtering never inspects status. A late native page failure retains consumed Inventory.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListStatusAndPageFailuresKeepPartialInventory`.
- Present filters, including explicit null or malformed JSON, invalid headers and nil options fail before HTTP; nil clears an earlier filter. A cancelling option stops later callbacks and HTTP while retaining the caller cause.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`, `image/image_record_cloud_test.go:TestCloudImageRecordOptionsCancelBeforeLaterCallbacks`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- The Go image Service is Glance v2 only, so supports_version(image, '2') is always the v2 branch. This owned profile requires paired Unicode JSON status strings; Python str.lower() also accepts lone surrogates.

### `python:cloud/_image/ImageCloudMixin/search_images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.SearchImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- SearchImageRecords completes the default list_images inventory, including the deleted-status filter and no member query, before the shared Cloud identifier exact-or-fnmatch phase. No name or filter is forwarded as a server query or Proxy Body filter; deleted rows are never selectable.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`, `connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Ordered dictionary filters inspect the 65 declared Image fields plus location, including nested location dictionaries; declared null attributes match while an unknown top-level key is a local error after inventory with retained Inventory. Falsey filters keep the identifier selection.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`.
- A JSON string filter runs the shared JMESPath engine on the selected views and returns arbitrary JSON in Value without inventing Images. Show_all/filter_deleted arguments, malformed filters and nil options fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`, `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- Python logs and ignores bad fnmatch patterns and raises AttributeError for unknown filter keys; Go uses the shared cloudfilter identifier and mapping contracts and returns an invalid-option error carrying the local cause.

### `python:cloud/_image/ImageCloudMixin/get_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- Omitted or JSON-null filters delegate to the owned Proxy find_image engine with ignore_missing true: literal GET, clean 400/403/404 fallback to the visible and hidden name lists, no Cloud deleted-status filter, nil Value/Image when missing and terminal native errors. Value is the found declared view.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetFindPathAndMissing`, `connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Every other filter value, including falsey JSON, uses search_images and then Python truthiness, len and index-0 selection. More than one element, including string code-point length, is ImageRecordSelectionError wrapping ErrAmbiguous; objects have no index 0; a selected JSON null or empty result is missing; Image is set only for one ordinary selected row while expression values stay raw.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetFilteredSelection`, `connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Filter_deleted/show_all arguments and an invalid find identity fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- The find path validates nonblank control-free identities like FindImageRecord. Unlike Cloud get_flavor, falsey non-null filters select the search path because the pinned source tests filters is not None.

### `python:cloud/_image/ImageCloudMixin/get_image_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecordByID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- GetImageRecordByID delegates to one strict literal GetImageRecord without query, Find fallback, missing-as-nil or deleted filtering; native 404 remains an error named GetImageRecordByID without nesting the delegated operation, and the Connection location is projected.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetByIDIsStrictLiteralGet`, `connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Only headers are accepted. Filters, show_all, blank IDs and nil options fail before HTTP, and each option runs as one checked Get option so cancellation stops later callbacks.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`, `image/image_record_cloud_test.go:TestCloudImageRecordOptionsCancelBeforeLaterCallbacks`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- Go accepts a literal string ID only and rejects blank, dot-segment and control identities before HTTP; it does not accept a Resource seed or Attributes like the Proxy GetImageRecord request.

### `python:cloud/_image/ImageCloudMixin/get_image_exclude`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecordExclude`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- The helper completes the default filter-free search_images inventory first, so deleted rows are never candidates and an empty identifier selects every row. An empty exclude returns the first selected row; otherwise rows are visited in order and the first whose name does not contain exclude is selected without inspecting later rows.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`, `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`.
- Exclude membership follows Python `in` on the raw name: substring for strings, string element equality for lists and key membership for dictionaries. A reached null, number or bool name fails with the actual list-page ResponseError and retained Inventory, while unreached rows are not inspected. Absent selection returns nil Image and Value with the consumed Inventory.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.
- Filters, show_all, filter_deleted and invalid UTF-8 exclude fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeArgumentsFailBeforeHTTP`.
- Value is the selected row's declared Resource view, including Connection location.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.

Python/Go differences:

- Python returns a connected mutable Image Resource or its raw attribute; Go returns an owned CloudImageRecordResult whose Value is the Source return value and whose Image keeps actual Wire/Envelope/header/status evidence. No Resource identity, adapter cache or session parity is claimed.
- The owned profile requires paired Unicode JSON name strings for exclude membership, while Python str also admits lone surrogates. Headers are a Go extension for every request; filters and list controls are absent from the Source signature and fail before HTTP.

### `python:cloud/_image/ImageCloudMixin/get_image_name`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecordName`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- The helper completes the default filter-free search_images inventory first, so deleted rows are never candidates and an empty identifier selects every row. An empty exclude returns the first selected row; otherwise rows are visited in order and the first whose name does not contain exclude is selected without inspecting later rows.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`, `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`.
- Exclude membership follows Python `in` on the raw name: substring for strings, string element equality for lists and key membership for dictionaries. A reached null, number or bool name fails with the actual list-page ResponseError and retained Inventory, while unreached rows are not inspected. Absent selection returns nil Image and Value with the consumed Inventory.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.
- Filters, show_all, filter_deleted and invalid UTF-8 exclude fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeArgumentsFailBeforeHTTP`.
- Value is the selected row's raw name JSON; a selected null or list name remains present and distinct from no selection.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.

Python/Go differences:

- Python returns a connected mutable Image Resource or its raw attribute; Go returns an owned CloudImageRecordResult whose Value is the Source return value and whose Image keeps actual Wire/Envelope/header/status evidence. No Resource identity, adapter cache or session parity is claimed.
- The owned profile requires paired Unicode JSON name strings for exclude membership, while Python str also admits lone surrogates. Headers are a Go extension for every request; filters and list controls are absent from the Source signature and fail before HTTP.

### `python:cloud/_image/ImageCloudMixin/get_image_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecordID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- The helper completes the default filter-free search_images inventory first, so deleted rows are never candidates and an empty identifier selects every row. An empty exclude returns the first selected row; otherwise rows are visited in order and the first whose name does not contain exclude is selected without inspecting later rows.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`, `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`.
- Exclude membership follows Python `in` on the raw name: substring for strings, string element equality for lists and key membership for dictionaries. A reached null, number or bool name fails with the actual list-page ResponseError and retained Inventory, while unreached rows are not inspected. Absent selection returns nil Image and Value with the consumed Inventory.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.
- Filters, show_all, filter_deleted and invalid UTF-8 exclude fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeArgumentsFailBeforeHTTP`.
- Value is the selected row's raw id JSON; the identifier phase matches numeric IDs by their string representation.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordExcludeMembershipOrderAndValues`.

Python/Go differences:

- Python returns a connected mutable Image Resource or its raw attribute; Go returns an owned CloudImageRecordResult whose Value is the Source return value and whose Image keeps actual Wire/Envelope/header/status evidence. No Resource identity, adapter cache or session parity is claimed.
- The owned profile requires paired Unicode JSON name strings for exclude membership, while Python str also admits lone surrogates. Headers are a Go extension for every request; filters and list controls are absent from the Source signature and fail before HTTP.

### `python:cloud/_image/ImageCloudMixin/wait_for_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.WaitForCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitUnlimited`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudWaitPollInterval`

- WaitForCloudImageRecord reads only the supplied record's string id and ignores its cached status. Each iteration repeats the owned Proxy find_image lookup with ignore_missing true and shared headers/location: a missing image, case-different status, null or non-string status keeps polling, exact active returns that record and exact error returns FailedStateError with the last actual receipt.
  Tests: `image/image_record_cloud_wait_test.go:TestCloudImageRecordWaitPollsFindUntilExactActive`, `image/image_record_cloud_wait_test.go:TestCloudImageRecordWaitFailuresKeepLastRecord`.
- The iterate_timeout policy defaults to 3600 seconds and a two-second interval, checks the deadline before each lookup, expires zero or negative timeouts before HTTP and supports unlimited waiting. Timeout, error state and caller cancellation during the sleep return Lookups and the last found record with its receipt; a terminal find error stops polling immediately.
  Tests: `image/image_record_cloud_wait_test.go:TestCloudImageRecordWaitFailuresKeepLastRecord`.
- Nil, numeric or blank ids, nil options, nonpositive poll intervals and contradictory unlimited/timeout options fail before HTTP.
  Tests: `image/image_record_cloud_wait_test.go:TestCloudImageRecordWaitInputFailsBeforeHTTP`.

Python/Go differences:

- Python accepts any mapping with an id and raises ResourceTimeout or SDKException; Go accepts an owned ImageRecord with a nonblank string id and returns ImageRecordCloudWaitResult with context.DeadlineExceeded or FailedStateError plus the last actual receipt.
- The fixed two-second Python interval is the default; a positive PollInterval and request headers are Go extensions. An in-flight lookup is not cancelled by the wait deadline, matching iterate_timeout, while caller context cancellation is observed during sleeps.

### `python:cloud/_image/ImageCloudMixin/download_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DownloadCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadStream`

- Missing or duplicate destinations, typed nil writers, invalid filenames, blank identities, nil options and invalid headers fail before HTTP.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadStrictFindAndPreflight`.
- DownloadCloudImageRecord finds the image with the owned ignore_missing false Find graph, returning ErrNotFound without download requests when absent, then passes the found record to DownloadImageRecord: mandatory metadata GET of the found identity, binary GET of the fetched identity and default checksum verification, writing to the borrowed writer or SDK-created file. Stream is forwarded but the required destination wins; chunk size and headers reach the lower phases.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadFindsThenDownloadsToRequiredOutput`, `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadStrictFindAndPreflight`.
- A lower download failure returns Found plus the partial download result and the native cause under the DownloadCloudImageRecord operation without nesting the delegated operation.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadFindsThenDownloadsToRequiredOutput`.

Python/Go differences:

- Python returns the lower download value or raises SDKException/ResourceNotFound; Go returns ImageRecordCloudDownloadResult separating the strict Find record from the lower download phases and wraps typed errors.
- Python treats an empty output_path as present and then falls back to memory mode; Go treats an empty Filename as absent and rejects it before HTTP. Headers are a Go extension for both phases, and Proxy-only download options keep their defaults because the Cloud signature has no such arguments.

### `python:cloud/_image/ImageCloudMixin/update_image_properties`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateCloudImageProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperty`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertyMeta`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesMetaJSON`

- UpdateCloudImageProperties selects image or name_or_id: a Record wins over both string forms, otherwise a nonempty literal ID wins over NameOrID, and NameOrID is passed on as a literal identity without any name lookup or HTTP.
  Tests: `image/image_record_cloud_properties_test.go:TestCloudImagePropertiesSelectsImageOrNameOrIDAndDelegates`.
- The selected image is delegated unchanged to the owned UpdateImagePropertiesRecord helper with the same options, returning its Updated/Record result: cached property copy, kernel/ramdisk lookup, conversions, raw meta precedence and conditional automatic PATCH. Literal identities keep the lower helper's missing cached-properties failure before HTTP, and errors are renamed to the Cloud operation without nesting.
  Tests: `image/image_record_cloud_properties_test.go:TestCloudImagePropertiesSelectsImageOrNameOrIDAndDelegates`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed`, `image/image_record_properties_test.go:TestImageRecordPropertiesRawValuesPythonStringsAndMetaLast`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting`.

Python/Go differences:

- Python's image argument may be an Image object or a string; Go separates Record and ID fields and keeps NameOrID for name_or_id. Python truthiness of an empty string is represented by empty Go strings.
- Properties and meta use the existing concrete ImageRecordPropertiesOption values instead of Python **properties and meta dictionaries; the lower helper's documented Go differences apply unchanged.

### `python:cloud/_image/ImageCloudMixin/delete_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DeleteCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteWait`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeleteUnlimited`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDeletePollInterval`

- DeleteCloudImageRecord finds the image with the ignore-missing Find graph; absence returns Deleted false without DELETE. A found record is deleted with the owned whole-image DeleteImageRecord, and a native delete failure returns Found under the Cloud operation before any Swift cleanup or wait.
  Tests: `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteFindsDeletesCleansAndWaits`, `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteLateFailuresKeepPhases`.
- When the captured image_api_use_tasks value is Python-truthy, the found properties are checked for the openstack object key and then the shade key; the stored container/name is split at the first slash and deleted through objects.DeleteObject with SLO HEAD discovery and ignore-missing defaults, without image headers. Falsey or absent configuration and missing keys skip Swift.
  Tests: `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteFindsDeletesCleansAndWaits`.
- Listed null properties, non-string or null object locations and values without a separator fail after the image delete with the found record's receipt and no Swift request. Optional wait repeats the shared Cloud find loop until absence, and timeout keeps the last found record.
  Tests: `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteLateFailuresKeepPhases`, `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteFindsDeletesCleansAndWaits`.
- Blank identities, nil options, nonpositive intervals, invalid headers and invalid captured task configuration fail before HTTP.
  Tests: `image/image_record_cloud_delete_test.go:TestCloudImageRecordDeleteInputFailsBeforeHTTP`.

Python/Go differences:

- Python returns a bool and raises SDKException, TypeError, AttributeError, ValueError or ResourceTimeout; Go returns ImageRecordCloudDeleteResult with Deleted, Found, Delete, Object and wait evidence plus typed errors.
- The pinned delete_objects argument is never read by the source body, so Go omits it. Wait polling intervals and request headers are Go extensions; headers apply to Glance requests only.

### `python:cloud/_image/ImageCloudMixin/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.PrepareImageRecordCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateOpts`

- Without a volume CreateCloudImageRecord runs the owned CreateImageRecord graph with one prepared option snapshot and returns its Record when wait is falsey; with a volume it maps truthy allow_duplicates to force and Python-falsey container/disk formats to the Cinder Proxy defaults before CreateVolumeImageRecord.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`, `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.
- A truthy wait repeats Cloud get_image on the created id: missing images keep polling, a found status other than exact queued or saving, including null, is returned, and WaitLookups/WaitLast keep the evidence.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- The iterate_timeout deadline uses the raw create timeout (default 3600 seconds, null unlimited); on timeout the image is deleted through Cloud delete_image with wait and the DeadlineExceeded error keeps the cleanup result and created record.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- Truthy non-string volume formats, non-numeric timeouts with wait, nil options and negative poll intervals fail before HTTP.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.

Python/Go differences:

- Python returns an Image or None and raises ResourceTimeout or the cleanup exception; Go returns CloudImageCreateResult with Created, Volume, wait and Cleanup evidence and joins the timeout with any cleanup failure.
- Python keyword arguments become the concrete ImageRecordCreateOption values; the volume is a literal VolumeID and PollInterval is a Go extension for the Cloud wait and its cleanup.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
visible = conn.list_images()
everything = conn.list_images(show_all=True)
ubuntu = conn.search_images("ubuntu-*", filters={"visibility": "public"})
image = conn.get_image("ubuntu-24.04")  # find_image, ignore_missing=True
by_id = conn.get_image_by_id("4c1e7d5a-0b6f-4a28-9a5e-7b0f3c1d2e9f")
print(len(visible), len(everything), [i.id for i in ubuntu])
if image is not None:
    print(image.to_dict(), by_id.status)
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "image name or ID glob")
    showAll := flag.Bool("show-all", false, "include shared images that are not accepted")
    identity := flag.String("image", "", "optional image name or ID for Cloud get_image")
    id := flag.String("id", "", "optional image ID for strict get_image_by_id")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *showAll, *identity, *id); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *image.ImageRecord) any {
    if record == nil {
        return nil
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "import_methods": record.ImportMethods,
    }
}

func printQuery(operation string, result *image.ImageRecordQueryResult) error {
    if result == nil {
        return nil
    }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "image_count": len(result.Images), "inventory_count": len(result.Inventory),
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern string, showAll bool, identity, id string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Image(ctx)
    if err != nil {
        return err
    }
    all, err := service.AllCloudImageRecords(ctx, image.WithImageRecordQueryShowAll(showAll))
    if printErr := printQuery("list_images", all); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("list images: %w", err)
    }
    matched, err := service.SearchImageRecords(ctx, pattern)
    if printErr := printQuery("search_images", matched); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("search images: %w", err)
    }
    if identity != "" {
        found, err := service.GetCloudImageRecord(ctx, identity)
        if err != nil {
            return fmt.Errorf("get image: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image", "value": found.Value, "record": recordValue(found.Image)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    if id != "" {
        record, err := service.GetImageRecordByID(ctx, id)
        if err != nil {
            return fmt.Errorf("get image by id: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image_by_id", "record": recordValue(record)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    return nil
}
```

## References

- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Record Download](image-record-download.md)
- [Image Record Properties](image-record-properties.md)
- [Image Record Delete](image-record-delete.md)
- [Image Record Create](image-record-create.md)
- [Cinder Proxy create_image](../blockstorage/volume-upload-image.md)
- [Cloud list_images·search_images·get_image·get_image_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L166)
- [get_image_exclude·get_image_name·get_image_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)
- [wait_for_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L248-L263)
- [download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L168-L219)
- [update_image_properties](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L432-L445)
- [delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L265-L313)
- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L315-L430)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
