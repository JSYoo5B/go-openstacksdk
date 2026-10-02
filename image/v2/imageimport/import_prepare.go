package imageimport

import "context"

// PrepareImportOptions validates the selected source, freezes caller options,
// and checks configured store-header conflicts without making any request.
// Pass the returned policy to WithImportOpts to compose prevalidated phases.
func (a *API) PrepareImportOptions(ctx context.Context, options ...ImportOption) (ImportOpts, error) {
	if a == nil {
		return ImportOpts{}, wrapImportError(ctx, importInvalid("image service client is required"))
	}
	if err := validateImportSource(ctx, a.client); err != nil {
		return ImportOpts{}, wrapImportError(ctx, err)
	}
	policy, err := parseImportOpts(append([]ImportOption(nil), options...))
	if err == nil {
		err = validateImportSource(ctx, a.client)
	}
	if err == nil {
		_, err = preparedImportHeaders(a.client.MoreHeaders, a.client.Microversion, policy)
	}
	return policy, wrapImportError(ctx, err)
}

func preparedImportHeaders(source map[string]string, version string, policy ImportOpts) (map[string]string, error) {
	headers, err := importHeaders(source, true, version)
	if err != nil {
		return nil, err
	}
	for key, value := range policy.Headers {
		headers[key] = value
	}
	if policy.Store != nil {
		headers["X-Image-Meta-Store"] = *policy.Store
	}
	if _, exists := headers["X-Image-Meta-Store"]; exists && (len(policy.Stores) > 0 || policy.AllStores != nil && *policy.AllStores) {
		return nil, importInvalid("source store header conflicts with Stores or AllStores")
	}
	return headers, nil
}
