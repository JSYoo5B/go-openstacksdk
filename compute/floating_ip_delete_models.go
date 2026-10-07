package compute

import "fmt"

// Each logical attempt retains an accepted DELETE or actual failure, followed
// by its public-Get verification. Native transport retries are not new entries.
type FloatingIPDeleteAttempt struct {
	Backend      FloatingIPSource
	Accepted     bool
	Response     *FloatingIPQueryResponse
	Failure      *FloatingIPQueryResponse
	NotFound     error
	Verification *GetFloatingIPResult
	Verified     bool
	Absent       bool
	Down         bool
}

// Deleted is cloud's final boolean result. DOWN can satisfy that policy while
// the row is still present; Absent only records successful lookup with no row.
// On error, accepted attempts remain observable without asserting deletion.
type DeleteFloatingIPResult struct {
	ID               string
	Backend          FloatingIPSource
	Deleted          bool
	Absent           bool
	Down             bool
	Attempts         []*FloatingIPDeleteAttempt
	LastVerification *GetFloatingIPResult
	Failure          *FloatingIPQueryResponse
}

type FloatingIPDeleteVerificationError struct {
	ID         string
	Attempts   int
	FloatingIP *FloatingIPRecord
}

func (e *FloatingIPDeleteVerificationError) Error() string {
	return fmt.Sprintf("floating IP %q remains after %d delete attempts", e.ID, e.Attempts)
}
