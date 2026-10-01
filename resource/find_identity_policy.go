package resource

// FindFallbackPolicy selects when an identity lookup may follow a failed
// direct GET with a list search. It does not change explicit ID/Name Ref lookup.
type FindFallbackPolicy int

const (
	// FindFallbackCompatible follows the pinned Python Resource.find policy:
	// direct GET responses 400, 403 and 404 may fall back to listing.
	FindFallbackCompatible FindFallbackPolicy = iota
	// FindFallbackNotFoundOnly permits list fallback only after HTTP 404.
	FindFallbackNotFoundOnly
	// FindFallbackNever keeps identity lookup to its direct GET.
	FindFallbackNever
)
