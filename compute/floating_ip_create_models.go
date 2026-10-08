package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/network"
)

// Allocation and AllocationResponse remain the initial mutation evidence.
// FloatingIP is the final successful view or last known allocation; waiting
// observations and cleanup are separate and never erase an accepted mutation.
type CreateFloatingIPResult struct {
	Backend                FloatingIPSource
	Allocated, Waited      bool
	FloatingIP, Allocation *FloatingIPRecord
	AllocationResponse     *FloatingIPQueryResponse
	Selection              network.FloatingIPSelection
	PoolQuery              *FloatingIPPoolQueryResult
	Compatibility          *GetFloatingIPResult
	Observations           []*GetFloatingIPResult
	FallbackError          error
	Failure                *FloatingIPQueryResponse
	Cleanup                *DeleteFloatingIPResult
	CleanupError           error
}

type FloatingIPCreateTimeoutError struct {
	ID      string
	Timeout time.Duration
	Last    *FloatingIPRecord
}

func (e *FloatingIPCreateTimeoutError) Error() string {
	return fmt.Sprintf("floating IP %q did not become ACTIVE within %s", e.ID, e.Timeout)
}
func (e *FloatingIPCreateTimeoutError) Unwrap() error { return context.DeadlineExceeded }
