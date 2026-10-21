package resource

// IgnoresMissing evaluates lookup options for operations that apply the common
// ignore-missing policy outside a Collection. The default applies when no option
// selects a policy; later options win and a nil option is rejected.
func IgnoresMissing(defaultIgnore bool, options ...LookupOption) (bool, error) {
	o, err := parseLookup(defaultIgnore, options)
	return o.ignoreMissing, err
}
