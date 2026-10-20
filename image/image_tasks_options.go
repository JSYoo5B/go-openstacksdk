package image

// ListImageTasksOpts supplies ordinary headers and a local row cap. Zero
// MaxItems is unlimited; no limit or pagination query is sent to the server.
type ListImageTasksOpts struct {
	Headers  map[string]string
	MaxItems int
}
type ListImageTasksOption func(*ListImageTasksOpts) error

// WithListImageTasksOpts snapshots and replaces the entire configuration.
func WithListImageTasksOpts(value ListImageTasksOpts) ListImageTasksOption {
	snapshot := copyListImageTasksOpts(value)
	return func(config *ListImageTasksOpts) error { *config = copyListImageTasksOpts(snapshot); return nil }
}
func WithListImageTasksHeader(key, value string) ListImageTasksOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ListImageTasksOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithListImageTasksHeaders(value map[string]string) ListImageTasksOption {
	apply := WithImageMutationHeaders(value)
	return func(config *ListImageTasksOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithListImageTasksMaxItems(value int) ListImageTasksOption {
	return func(config *ListImageTasksOpts) error { config.MaxItems = value; return nil }
}

func copyListImageTasksOpts(value ListImageTasksOpts) ListImageTasksOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	return value
}
func parseListImageTasksOptions(options []ListImageTasksOption) (ListImageTasksOpts, error) {
	value, err := applyTaskOptions(ListImageTasksOpts{}, options, copyListImageTasksOpts)
	if err != nil {
		return value, err
	}
	if value.MaxItems < 0 {
		return value, uploadInvalid("image task max items must be non-negative")
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, err
}
