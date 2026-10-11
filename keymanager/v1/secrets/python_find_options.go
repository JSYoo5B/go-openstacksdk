package secrets

import "github.com/JSYoo5B/go-openstacksdk/resource"

// FindRecordOpts owns the finite find_secret controls. IgnoreMissing defaults
// to true; Fallback defaults to compatible clean HTTP400/403/404 fallback.
// The SDK always checks HTTP status, including Secret's custom payload fetch.
type FindRecordOpts struct {
	IgnoreMissing *bool
	Fallback      resource.FindFallbackPolicy
}

type FindRecordOption func(*FindRecordOpts) error

func ownFindRecordOptions(value FindRecordOpts) FindRecordOpts {
	if value.IgnoreMissing != nil {
		copy := *value.IgnoreMissing
		value.IgnoreMissing = &copy
	}
	return value
}

func WithFindRecordOptions(value FindRecordOpts) FindRecordOption {
	snapshot := ownFindRecordOptions(value)
	return func(options *FindRecordOpts) error {
		*options = ownFindRecordOptions(snapshot)
		return nil
	}
}
func WithFindRecordIgnoreMissing(value bool) FindRecordOption {
	return func(options *FindRecordOpts) error {
		copy := value
		options.IgnoreMissing = &copy
		return nil
	}
}
func WithFindRecordFallback(value resource.FindFallbackPolicy) FindRecordOption {
	return func(options *FindRecordOpts) error {
		options.Fallback = value
		return nil
	}
}
