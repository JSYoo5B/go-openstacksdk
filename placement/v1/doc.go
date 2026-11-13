// Package v1 is the Placement v1 service: Connection.PlacementV1 returns a
// Service whose fields expose resource providers, inventories, allocations,
// allocation candidates, resource classes, traits and usages over Gophercloud's
// placement/v1 calls. WaitForStatus and WaitForDelete add Python-style waits.
//
// Calls send the service client's OpenStack-API-Version placement header
// unchanged; build the client with the microversion an operation needs.
// Python openstacksdk mappings and their differences are recorded in
// api/sdk_reviews.json.
package v1
