package serviceinfo

// WithStoreRecordListDetails selects Source stores(details=True)'s fixed detail
// route. It changes no query key, schema, role gate or native typed API behavior.
func WithStoreRecordListDetails(value bool) StoreRecordListOption {
	return func(c *StoreRecordListOpts) error { c.Details = value; return nil }
}
