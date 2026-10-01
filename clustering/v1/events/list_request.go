package events

import "gophercloudsdk/internal/senlin"

// WithListHeader adds a header to this list operation. Authentication, version
// and transport headers remain owned by the SDK.
func WithListHeader(key, value string) ListOption {
	return senlin.WithListHeader[ListOpts](key, value)
}

// WithListMicroversion selects a numeric Senlin version for this list operation
// without changing the shared service client.
func WithListMicroversion(value string) ListOption {
	return senlin.WithListMicroversion[ListOpts](value)
}
