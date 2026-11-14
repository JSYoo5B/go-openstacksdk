// Package accounts reads and changes Swift account metadata.
// Accounts.GetMetadata takes GetMetadataOpts and returns metadata, nullable
// counters and the raw HEAD204 response. Accounts.SetMetadata and
// Accounts.DeleteMetadata return the POST204 acknowledgement without reading
// again; SetMetadataAndRefresh adds the HEAD that Python performs by default.
package accounts
