// Package v1 is the Barbican key manager v1 service. Connection.KeyManagerV1
// returns a Service whose Secrets, Containers, Orders, Acls, SecretACLs,
// SecretConsumers, SecretStores, Quotas and Waits fields cover the service.
//
// Secrets, Containers and Orders add SDK-owned calls next to the native
// ones. Secrets.Fetch, Containers.Fetch and Orders.Fetch return the metadata
// GET with its HTTP evidence. Secrets.CreateRecord, Containers.CreateRecord and
// Orders.CreateRecord merge creation inputs, set with WithCreateRecordAttribute,
// with the POST response. Secrets.Remove, Containers.Remove and Orders.Remove
// delete and ignore a 404 unless resource.WithMissingError is given.
// ListRecords and AllRecords list records with local filters and marker or
// offset paging. None of them
// follows response references or fetches again after a write. SecretStores
// and Quotas are SDK-owned; Waits polls concrete secrets, containers, orders
// and secret stores.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v1
