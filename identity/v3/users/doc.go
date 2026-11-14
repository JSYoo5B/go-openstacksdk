// Package users manages Keystone users, their passwords and group membership.
// ListProjectRecords and ListGroupRecords return SDK-owned records for a fixed
// user ID, with the pinned UserProject query aliases and four local body
// filters on projects; native ListProjects and ListGroups keep their own
// arguments and pagers.
package users
