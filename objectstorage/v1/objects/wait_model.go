package objects

// ObjectWaitPhaseResult retains the physical requests of the latest HEAD
// poll, including native retries and any actual response or failure.
type ObjectWaitPhaseResult = ObjectCreatePhaseResult

// ObjectWaitResult keeps bounded, last-poll evidence. Complete means the
// requested condition was observed; Deleted requires a clean physical 404.
type ObjectWaitResult struct {
	Last              *ObjectWaitPhaseResult
	Polls             int
	Status            *string
	Complete, Deleted bool
}
