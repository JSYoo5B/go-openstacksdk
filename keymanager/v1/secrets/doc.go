// Package secrets manages Barbican secrets.
//
// GetPayload returns the whole payload as bytes, with Accept text/plain unless
// GetPayloadOpts selects another type. Fetch reads metadata and then the
// payload at the same ID when a content type is selected: the metadata default
// type, FetchOpts.ContentType, or none with FetchOpts.Payload false. Only an
// exact text/plain payload becomes text, and metadata survives a later payload
// failure.
//
// FindIdentity tries a metadata GET for a safe ID and, after a 400, 403 or 404,
// searches all metadata for an exact ID or name; resource.WithIdentityFindIgnoreMissing
// and resource.WithIdentityFindFallback change those policies. FindRecord also fetches a
// required payload. Resources.List and All classify Python filter names into
// 12 server queries and 12 local body fields, mapping algorithm to alg.
package secrets
