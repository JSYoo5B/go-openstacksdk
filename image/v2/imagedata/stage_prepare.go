package imagedata

import "context"

// PrepareStageOptions validates the selected source and freezes caller options
// without making requests or reading data. The returned value can be passed to
// WithStageOpts when composing a workflow that validates all phases first.
func (a *API) PrepareStageOptions(ctx context.Context, options ...StageOption) (StageOpts, error) {
	if a == nil {
		return StageOpts{}, wrapStageError(ctx, stageInvalid("image service client is required"))
	}
	if err := validateStageSource(ctx, a.client); err != nil {
		return StageOpts{}, wrapStageError(ctx, err)
	}
	policy, err := parseStageOpts(append([]StageOption(nil), options...))
	if err == nil {
		err = validateStageSource(ctx, a.client)
	}
	return policy, wrapStageError(ctx, err)
}
