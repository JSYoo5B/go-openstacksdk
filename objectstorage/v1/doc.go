// Package v1 is the Swift object storage v1 service. Connection.ObjectStorageV1
// returns a Service whose Accounts, Containers and Objects fields wrap
// Gophercloud's Swift calls and add SDK-owned metadata, transfer and wait
// workflows; the swauth package covers TempAuth authentication.
//
// GetInfo derives /info from the catalog endpoint and returns capabilities and
// raw extension JSON; GetInfoOpts adds headers. GetObjectSegmentSize picks the
// default 1 GiB or an explicit size within the advertised bounds; only a clean
// 404/412 falls back to the maximum of 2684354561 bytes (ObjectSegmentSizeOpts).
//
// GenerateFormSignature and GenerateTempURL sign with FormSignatureInput,
// GenerateFormSignatureOpts and GenerateTempURLOpts: SHA1 by default, SHA256 or
// SHA512 on request, integer UTC expiry. An explicit key needs no HTTP;
// otherwise form signing looks up the container key and then the account key,
// and temp URLs use the account key. GetTempURLKey (GetTempURLKeyOpts) reads
// the keys with HEAD204, the container before the account and the secondary
// key before the primary within each scope; a discovery error stops the
// fallback. Accounts.SetTempURLKey and Containers.SetTempURLKey take
// SetTempURLKeyOpts and send one POST204.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v1
