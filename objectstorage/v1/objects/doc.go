// Package objects manages Swift objects.
//
// Objects.CreateObject takes bytes, a file or a Reader through
// CreateObjectInput. Bytes use a single PUT; files and Readers are spooled,
// checksummed and compared with a HEAD before the call skips the upload or
// selects SLO/DLO segments under SegmentPrefix, keeping every attempt, manifest
// and cleanup response. IsObjectStale compares a HEAD with a local hash.
// Objects.CreateDirectoryMarkerObject writes an empty object with Content-Type
// application/directory. Native Create sends an empty body for nil Content.
//
// Objects.GetObject returns bytes, metadata and raw headers from one GET;
// Objects.DownloadObject writes to an io.Writer and Objects.StreamObject returns
// an io.ReadCloser the caller closes. ObjectReadOpts sets conditional headers,
// ranges and queries and tells 200/206/304 apart.
//
// Objects.GetMetadata reads with HEAD200. Objects.SetMetadata and
// Objects.DeleteMetadata HEAD with symlink=get, merge the change and POST202;
// MetadataChangeResult keeps both responses. HeadMetadata, SetMetadataFromFresh
// and DeleteMetadataAndRefresh follow the Python setter with a final HEAD.
//
// Objects.DeleteObject takes DeleteObjectOpts and by default sends HEAD before
// deleting the same object; a known SLO flag skips the HEAD, and true sends
// multipart-manifest=delete. Discovery and Deletion hold each phase's response,
// ordinary deletes accept 202/204, and only a clean 404 becomes IgnoredMissing.
//
// Objects.WaitForDelete sends HEAD every two seconds for up to 120 seconds
// until a clean 404. WaitForStatus compares a caller-selected attribute or
// header with no default timeout.
package objects
