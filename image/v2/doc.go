// Package v2 is the Glance image v2 service. Connection.ImageV2 returns a
// Service whose fields wrap Gophercloud's Glance calls and add SDK-owned task,
// staging, import, service information and identity calls; the image package
// adds the higher-level image.Service workflows. Each package and method
// documents its own requests, defaults and evidence.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v2
