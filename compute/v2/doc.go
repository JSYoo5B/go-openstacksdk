// Package v2 is the Nova compute v2 service. Connection.ComputeV2 returns a
// Service whose fields wrap Gophercloud's Nova calls and add SDK-owned record,
// keypair, console, tag, quota and instance action calls.
//
// Servers.WaitForServer waits for ACTIVE for up to 120 seconds; WaitForState
// fails on the exact ERROR or error status with no SDK timeout, and
// WaitForDelete defaults to 120 seconds, all polling every two seconds unless
// caller resource wait options say otherwise.
//
// Servers.FindIdentity and Flavors.FindIdentity resolve a name or ID string:
// GET first, then after a 400, 403 or 404 a list with exact ID and name
// matching and duplicate checks. For servers,
// resource.WithIdentityFindDetails(false) uses the summary list and
// resource.WithIdentityFindAllProjects(true) lists across projects. Flavor
// lookup scans every /flavors/detail page and fetches extra specs only with
// resource.WithIdentityFindExtraSpecs(true).
//
// Servers.GetPassword returns the encrypted password unless
// servers.WithGetPasswordPrivateKey supplies an RSA key. Tags.InServer resolves
// a server once and needs microversion 2.26. Limits.Fetch reads the current
// project's limits, and Connection.ProjectQuotas, CurrentProjectQuotas and
// ProjectLimits fix a project for quota and limit calls. The Nova image proxy
// record calls accept microversions up to 2.35, or 2.38 for metadata.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v2
