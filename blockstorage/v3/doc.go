// Package v3 is the Cinder block storage v3 service. Connection.BlockStorageV3
// returns a Service whose fields wrap Gophercloud's Cinder v3 calls and add
// SDK-owned record, metadata, group, message and administration calls.
//
// Volumes and Snapshots provide WaitForState, WaitForAvailable and
// WaitForDelete: state waits fail on the exact ERROR or error status and have
// no SDK timeout, deletion waits default to 120 seconds, and both poll every
// two seconds unless caller resource wait options say otherwise.
//
// Volumes.FindIdentity resolves a name or ID string: GET first, then after a
// 400, 403 or 404 a detailed list with exact ID and name matching and duplicate
// checks. resource.WithIdentityFindDetails(false) uses the summary list and
// resource.WithIdentityFindAllProjects(true) lists across projects.
//
// Volumes.MetadataIn and Snapshots.MetadataIn resolve a reference once and
// offer Get, Merge (POST), Replace (PUT) and DeleteKeys; a nil DeleteKeys
// deletes every key and an empty slice sends nothing. Limits.Fetch reads the
// current project's limits. Connection.BlockStorageProjectQuotas,
// CurrentBlockStorageProjectQuotas and BlockStorageProjectLimits fix a project
// for quota and limit calls; project limit filters need microversion 3.39.
//
// FindRecord on Volumes, Snapshots, Backups, Groups and GroupSnapshots, the
// Messages, Attachments and Transfers record calls and Volumes.ListSummary
// follow the Python Proxy shapes. Python openstacksdk mappings and differences
// are recorded in api/sdk_reviews.json.
package v3
