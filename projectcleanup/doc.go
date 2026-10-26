// Package projectcleanup plans and executes the deletion of resources owned by
// the current Keystone project, the Go counterpart of the Python cloud layer's
// project_cleanup.
//
// The workflow has two separate steps. Connection.PlanProjectCleanup only
// discovers resources and never sends a mutation; it returns a Plan with the
// project fixed at plan time, ordered stages of candidates, exclusions with
// their reasons and services that could not be examined. After inspecting the
// plan, Connection.ExecuteProjectCleanup deletes exactly the plan's candidates,
// in stage order, without discovering again, and returns one Outcome per
// candidate.
//
// Each mutation is sent at most once. An uncertain request is reported and
// never replayed, and cancelling the context only stops new requests: work
// already accepted by a service may continue there. See README.md for the
// state table, the safety rules and the current service coverage.
package projectcleanup
