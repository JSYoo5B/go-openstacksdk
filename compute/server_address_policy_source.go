package compute

// FloatingIPSource returns the policy's configured floating-IP source. An
// unprepared zero policy reports the default, FloatingIPNeutron.
func (p ServerAddressPolicy) FloatingIPSource() FloatingIPSource {
	if !p.ready {
		return FloatingIPNeutron
	}
	return p.options.source
}
